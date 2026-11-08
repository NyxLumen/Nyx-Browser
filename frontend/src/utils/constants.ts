const appConfig = window.Nyx || window.FileBrowser || {};

const name: string = appConfig.Name || "Nyx Browser";
const disableExternal: boolean = appConfig.DisableExternal;
const disableUsedPercentage: boolean = appConfig.DisableUsedPercentage;
const baseURL: string = appConfig.BaseURL;
const staticURL: string = appConfig.StaticURL;
const recaptcha: string = appConfig.ReCaptcha;
const recaptchaKey: string = appConfig.ReCaptchaKey;
const signup: boolean = appConfig.Signup;
const version: string = appConfig.Version;
const logoURL = `${staticURL}/img/logo.svg`;
const noAuth: boolean = appConfig.NoAuth;
const authMethod = appConfig.AuthMethod;
const logoutPage: string = appConfig.LogoutPage;
const loginPage: boolean = appConfig.LoginPage;
const theme: UserTheme = appConfig.Theme;
const enableThumbs: boolean = appConfig.EnableThumbs;
const resizePreview: boolean = appConfig.ResizePreview;
const enableExec: boolean = appConfig.EnableExec;
const tusSettings = appConfig.TusSettings;
const origin = window.location.origin;
const tusEndpoint = `/api/tus`;
const hideLoginButton = appConfig.HideLoginButton;

export {
  name,
  disableExternal,
  disableUsedPercentage,
  baseURL,
  logoURL,
  recaptcha,
  recaptchaKey,
  signup,
  version,
  noAuth,
  authMethod,
  logoutPage,
  loginPage,
  theme,
  enableThumbs,
  resizePreview,
  enableExec,
  tusSettings,
  origin,
  tusEndpoint,
  hideLoginButton,
};
