package cmd

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/asdine/storm/v3"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"

	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/storage"
	"github.com/nyxlumen/nyx-browser/v2/storage/bolt"
)

const databasePermissions = 0640

func getAndParseFileMode(flags *pflag.FlagSet, name string) (fs.FileMode, error) {
	mode, err := flags.GetString(name)
	if err != nil {
		return 0, err
	}

	b, err := strconv.ParseUint(mode, 0, 32)
	if err != nil {
		return 0, err
	}

	return fs.FileMode(b), nil
}

func generateKey() []byte {
	k, err := settings.GenerateKey()
	if err != nil {
		panic(err)
	}
	return k
}

func dbExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat.Size() != 0, nil
	}

	if os.IsNotExist(err) {
		d := filepath.Dir(path)
		_, err = os.Stat(d)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0700); err != nil {
				return false, err
			}
			return false, nil
		}
	}

	return false, err
}

// Generate the replacements for all environment variables. This allows to
// use NYX_BRANDING_DISABLE_EXTERNAL environment variables, even when the
// option name is branding.disableExternal.
func generateEnvKeyReplacements(cmd *cobra.Command) []string {
	replacements := []string{}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		oldName := strings.ToUpper(f.Name)
		newName := strings.ToUpper(lo.SnakeCase(f.Name))
		replacements = append(replacements, oldName, newName)
	})

	return replacements
}

func initViper(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()

	// Ensure persistent flags are merged into cmd.Flags()
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if cmd.Flags().Lookup(f.Name) == nil {
			cmd.Flags().AddFlag(f)
		}
	})

	// Get config file from flag
	cfgFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	// Environment variables
	v.SetEnvPrefix("NYX")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(generateEnvKeyReplacements(cmd)...))

	// Bind fallback FB_<KEY> environment variables so NYX_ takes strict precedence
	// when both are provided, and FB_ works when NYX_ is absent.
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		fbSnake := "FB_" + strings.ToUpper(lo.SnakeCase(f.Name))
		fbUpper := "FB_" + strings.ToUpper(f.Name)
		_ = v.BindEnv(f.Name, fbSnake, fbUpper)
	})

	// Bind the flags
	err = v.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	// Configuration file resolution:
	// If explicit cfgFile provided, load it directly.
	// Otherwise search for .nyx in (., home, /etc/nyx/, /etc/filebrowser/).
	// Fallback to .filebrowser in the same directories with an informative notice.
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			if errors.As(err, &viper.ConfigParseError{}) {
				return nil, err
			}
			log.Println("No config file used")
		} else {
			log.Printf("Using config file: %s", v.ConfigFileUsed())
		}
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		v.AddConfigPath(".")
		v.AddConfigPath(home)
		v.AddConfigPath("/etc/nyx/")
		v.AddConfigPath("/etc/filebrowser/")

		v.SetConfigName(".nyx")
		if err := v.ReadInConfig(); err != nil {
			if errors.As(err, &viper.ConfigParseError{}) {
				return nil, err
			}
			var notFoundErr viper.ConfigFileNotFoundError
			if errors.As(err, &notFoundErr) {
				v.SetConfigName(".filebrowser")
				if fallbackErr := v.ReadInConfig(); fallbackErr != nil {
					if errors.As(fallbackErr, &viper.ConfigParseError{}) {
						return nil, fallbackErr
					}
					log.Println("No config file used")
				} else {
					log.Printf("Using legacy config file: %s (consider renaming to .nyx.*)", v.ConfigFileUsed())
				}
			} else {
				log.Println("No config file used")
			}
		} else {
			log.Printf("Using config file: %s", v.ConfigFileUsed())
		}
	}

	// Return Viper
	return v, nil
}

// resolveDatabase determines the database path and whether it already exists.
// If the target is nyx.db and does not exist, but legacy filebrowser.db exists in the
// same directory, filebrowser.db is returned with an informative log message without
// modifying, renaming, or deleting any existing database.
func resolveDatabase(v *viper.Viper) (string, bool, error) {
	path, err := filepath.Abs(v.GetString("database"))
	if err != nil {
		return "", false, err
	}

	exists, err := dbExists(path)
	if err != nil {
		return "", false, err
	}

	if !exists && filepath.Base(path) == "nyx.db" {
		legacyPath := filepath.Join(filepath.Dir(path), "filebrowser.db")
		legacyExists, legacyErr := dbExists(legacyPath)
		if legacyErr == nil && legacyExists {
			log.Printf("Using legacy database: %s. No data will be modified or renamed.", legacyPath)
			return legacyPath, true, nil
		}
	}

	return path, exists, nil
}

type store struct {
	*storage.Storage
	databaseExisted bool
}

type storeOptions struct {
	expectsNoDatabase bool
	allowsNoDatabase  bool
}

type cobraFunc func(cmd *cobra.Command, args []string) error

// withViperAndStore initializes Viper and the storage.Store and passes them to the callback function.
// This function should only be used by [withStore] and the root command. No other command should call
// this function directly.
func withViperAndStore(fn func(cmd *cobra.Command, args []string, v *viper.Viper, store *store) error, options storeOptions) cobraFunc {
	return func(cmd *cobra.Command, args []string) error {
		v, err := initViper(cmd)
		if err != nil {
			return err
		}

		path, exists, err := resolveDatabase(v)
		if err != nil {
			return err
		}

		switch {
		case exists && options.expectsNoDatabase:
			log.Fatal(path + " already exists")
		case !exists && !options.expectsNoDatabase && !options.allowsNoDatabase:
			log.Fatal(path + " does not exist. Please run 'nyx config init' first.")
		case !exists && !options.expectsNoDatabase:
			log.Println("WARNING: nyx.db can't be found. Initialing in " + strings.TrimSuffix(path, "nyx.db"))
		}

		log.Println("Using database: " + path)

		db, err := storm.Open(path, storm.BoltOptions(databasePermissions, nil))
		if err != nil {
			return err
		}
		defer db.Close()

		storage, err := bolt.NewStorage(db)
		if err != nil {
			return err
		}

		store := &store{
			Storage:         storage,
			databaseExisted: exists,
		}

		return fn(cmd, args, v, store)
	}
}

func withStore(fn func(cmd *cobra.Command, args []string, store *store) error, options storeOptions) cobraFunc {
	return withViperAndStore(func(cmd *cobra.Command, args []string, _ *viper.Viper, store *store) error {
		return fn(cmd, args, store)
	}, options)
}

func marshal(filename string, data interface{}) error {
	fd, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	switch ext := filepath.Ext(filename); ext {
	case ".json":
		encoder := json.NewEncoder(fd)
		encoder.SetIndent("", "    ")
		return encoder.Encode(data)
	case ".yml", ".yaml":
		encoder := yaml.NewEncoder(fd)
		return encoder.Encode(data)
	default:
		return errors.New("invalid format: " + ext)
	}
}

func unmarshal(filename string, data interface{}) error {
	fd, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	switch ext := filepath.Ext(filename); ext {
	case ".json":
		return json.NewDecoder(fd).Decode(data)
	case ".yml", ".yaml":
		return yaml.NewDecoder(fd).Decode(data)
	default:
		return errors.New("invalid format: " + ext)
	}
}

func jsonYamlArg(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}

	switch ext := filepath.Ext(args[0]); ext {
	case ".json", ".yml", ".yaml":
		return nil
	default:
		return errors.New("invalid format: " + ext)
	}
}

// convertCmdStrToCmdArray checks if cmd string is blank (whitespace included)
// then returns empty string array, else returns the split word array of cmd.
// This is to ensure the result will never be []string{""}
func convertCmdStrToCmdArray(cmd string) []string {
	var cmdArray []string
	trimmedCmdStr := strings.TrimSpace(cmd)
	if trimmedCmdStr != "" {
		cmdArray = strings.Split(trimmedCmdStr, " ")
	}
	return cmdArray
}
