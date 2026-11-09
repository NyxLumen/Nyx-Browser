package share

import (
	"errors"
	"strings"
	"testing"
	"time"

	fberrors "github.com/nyxlumen/nyx-browser/v2/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBackend struct {
	links map[string]*Link
}

func newMockBackend() *mockBackend {
	return &mockBackend{links: make(map[string]*Link)}
}

func (m *mockBackend) All() ([]*Link, error) {
	var list []*Link
	for _, l := range m.links {
		cpy := *l
		list = append(list, &cpy)
	}
	return list, nil
}

func (m *mockBackend) FindByUserID(id uint) ([]*Link, error) {
	var list []*Link
	for _, l := range m.links {
		if l.UserID == id {
			cpy := *l
			list = append(list, &cpy)
		}
	}
	return list, nil
}

func (m *mockBackend) GetByHash(hash string) (*Link, error) {
	l, ok := m.links[hash]
	if !ok {
		return nil, fberrors.ErrNotExist
	}
	cpy := *l
	return &cpy, nil
}

func (m *mockBackend) GetPermanent(path string, id uint) (*Link, error) {
	for _, l := range m.links {
		if l.Path == path && l.UserID == id && l.Expire == 0 {
			cpy := *l
			return &cpy, nil
		}
	}
	return nil, fberrors.ErrNotExist
}

func (m *mockBackend) Gets(path string, id uint) ([]*Link, error) {
	var list []*Link
	for _, l := range m.links {
		if l.Path == path && l.UserID == id {
			cpy := *l
			list = append(list, &cpy)
		}
	}
	return list, nil
}

func (m *mockBackend) Save(s *Link) error {
	m.links[s.Hash] = s
	return nil
}

func (m *mockBackend) Delete(hash string) error {
	delete(m.links, hash)
	return nil
}

func (m *mockBackend) DeleteWithPathPrefix(prefix string, userID uint) error {
	prefix = strings.TrimRight(prefix, "/")
	for hash, l := range m.links {
		if l.UserID != userID {
			continue
		}
		if l.Path == prefix || strings.HasPrefix(l.Path, prefix+"/") {
			delete(m.links, hash)
		}
	}
	return nil
}

type orderedBackend struct {
	links []*Link
}

func (b *orderedBackend) All() ([]*Link, error) {
	res := make([]*Link, len(b.links))
	copy(res, b.links)
	return res, nil
}

func (b *orderedBackend) FindByUserID(id uint) ([]*Link, error) {
	var res []*Link
	for _, l := range b.links {
		if l.UserID == id {
			res = append(res, l)
		}
	}
	return res, nil
}

func (b *orderedBackend) GetByHash(hash string) (*Link, error) {
	for _, l := range b.links {
		if l.Hash == hash {
			return l, nil
		}
	}
	return nil, fberrors.ErrNotExist
}

func (b *orderedBackend) GetPermanent(path string, id uint) (*Link, error) {
	for _, l := range b.links {
		if l.Path == path && l.UserID == id && l.Expire == 0 {
			return l, nil
		}
	}
	return nil, fberrors.ErrNotExist
}

func (b *orderedBackend) Gets(path string, id uint) ([]*Link, error) {
	var res []*Link
	for _, l := range b.links {
		if l.Path == path && l.UserID == id {
			res = append(res, l)
		}
	}
	return res, nil
}

func (b *orderedBackend) Save(s *Link) error {
	b.links = append(b.links, s)
	return nil
}

func (b *orderedBackend) Delete(hash string) error {
	for i, l := range b.links {
		if l.Hash == hash {
			b.links = append(b.links[:i], b.links[i+1:]...)
			return nil
		}
	}
	return nil
}

func (b *orderedBackend) DeleteWithPathPrefix(path string, userID uint) error {
	return nil
}

func TestStorage_ConsecutiveExpiredLinks(t *testing.T) {
	past := time.Now().Unix() - 100
	future := time.Now().Unix() + 1000

	t.Run("All filters all consecutive expired links without skipping", func(t *testing.T) {
		backend := &orderedBackend{
			links: []*Link{
				{Hash: "exp1", Expire: past, Path: "/a", UserID: 1},
				{Hash: "exp2", Expire: past, Path: "/b", UserID: 1},
				{Hash: "valid", Expire: future, Path: "/c", UserID: 1},
			},
		}

		store := NewStorage(backend)
		all, err := store.All()
		require.NoError(t, err)

		require.Len(t, all, 1, "expected only 1 valid link returned")
		assert.Equal(t, "valid", all[0].Hash)
		assert.Len(t, backend.links, 1, "expected both expired links to be deleted in backend")
	})

	t.Run("FindByUserID filters consecutive expired links", func(t *testing.T) {
		backend := &orderedBackend{
			links: []*Link{
				{Hash: "exp1", Expire: past, Path: "/a", UserID: 1},
				{Hash: "exp2", Expire: past, Path: "/b", UserID: 1},
				{Hash: "perm", Expire: 0, Path: "/c", UserID: 1},
			},
		}

		store := NewStorage(backend)
		links, err := store.FindByUserID(1)
		require.NoError(t, err)

		require.Len(t, links, 1)
		assert.Equal(t, "perm", links[0].Hash)
		assert.Len(t, backend.links, 1)
	})

	t.Run("Gets filters consecutive expired links", func(t *testing.T) {
		backend := &orderedBackend{
			links: []*Link{
				{Hash: "exp1", Expire: past, Path: "/shared", UserID: 1},
				{Hash: "exp2", Expire: past, Path: "/shared", UserID: 1},
				{Hash: "valid", Expire: future, Path: "/shared", UserID: 1},
			},
		}

		store := NewStorage(backend)
		links, err := store.Gets("/shared", 1)
		require.NoError(t, err)

		require.Len(t, links, 1)
		assert.Equal(t, "valid", links[0].Hash)
		assert.Len(t, backend.links, 1)
	})
}

func TestStorage_CRUDAndExpiration(t *testing.T) {
	backend := newMockBackend()
	store := NewStorage(backend)

	link1 := &Link{
		Hash:   "hash1",
		Path:   "/docs/file1.pdf",
		UserID: 1,
		Expire: 0, // permanent
	}
	link2 := &Link{
		Hash:   "hash2",
		Path:   "/docs/file2.pdf",
		UserID: 1,
		Expire: time.Now().Unix() + 3600, // future
	}
	link3 := &Link{
		Hash:   "hash3",
		Path:   "/docs/sub/file3.pdf",
		UserID: 1,
		Expire: time.Now().Unix() - 10, // already expired
	}

	require.NoError(t, store.Save(link1))
	require.NoError(t, store.Save(link2))
	require.NoError(t, store.Save(link3))

	// GetByHash
	got1, err := store.GetByHash("hash1")
	require.NoError(t, err)
	assert.Equal(t, "/docs/file1.pdf", got1.Path)

	// GetByHash on expired link should delete it and return ErrNotExist
	got3, err := store.GetByHash("hash3")
	assert.True(t, errors.Is(err, fberrors.ErrNotExist))
	assert.Nil(t, got3)

	// GetPermanent
	perm, err := store.GetPermanent("/docs/file1.pdf", 1)
	require.NoError(t, err)
	assert.Equal(t, "hash1", perm.Hash)

	// Delete
	require.NoError(t, store.Delete("hash1"))
	_, err = store.GetByHash("hash1")
	assert.True(t, errors.Is(err, fberrors.ErrNotExist))

	// DeleteWithPathPrefix
	link4 := &Link{Hash: "hash4", Path: "/folder/sub/a.txt", UserID: 2, Expire: 0}
	link5 := &Link{Hash: "hash5", Path: "/folder/sub/b.txt", UserID: 2, Expire: 0}
	link6 := &Link{Hash: "hash6", Path: "/other/c.txt", UserID: 2, Expire: 0}
	require.NoError(t, store.Save(link4))
	require.NoError(t, store.Save(link5))
	require.NoError(t, store.Save(link6))

	require.NoError(t, store.DeleteWithPathPrefix("/folder/sub", 2))
	_, err = store.GetByHash("hash4")
	assert.True(t, errors.Is(err, fberrors.ErrNotExist))
	_, err = store.GetByHash("hash5")
	assert.True(t, errors.Is(err, fberrors.ErrNotExist))
	got6, err := store.GetByHash("hash6")
	require.NoError(t, err)
	assert.Equal(t, "/other/c.txt", got6.Path)
}
