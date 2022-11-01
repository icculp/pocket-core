package iavl

import (
	"fmt"
	"testing"

	rand2 "github.com/tendermint/tendermint/libs/rand"

	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tm-db"

	"github.com/pokt-network/pocket-core/store/types"
)

var (
	cacheSize        = 100
	numRecent  int64 = 5
	storeEvery int64 = 3
)

var (
	treeData = map[string]string{
		"hello": "goodbye",
		"aloha": "shalom",
	}
	nMoreData = 0
)

// make a tree with data from above and save it
func newAlohaTree(t *testing.T, db dbm.DB) (*MutableTree, types.CommitID) {
	tree, _ := NewMutableTree(db, cacheSize)
	for k, v := range treeData {
		tree.Set([]byte(k), []byte(v))
	}
	for i := 0; i < nMoreData; i++ {
		key := rand2.Bytes(12)
		value := rand2.Bytes(50)
		tree.Set(key, value)
	}
	hash, ver, err := tree.SaveVersion()
	require.Nil(t, err)
	return tree, types.CommitID{Version: ver, Hash: hash}
}

func TestTestGetImmutableIterator(t *testing.T) {
	db := dbm.NewMemDB()
	tree, cID := newAlohaTree(t, db)
	store := UnsafeNewStore(tree, 10, 10)

	newStore, err := store.LazyLoadStore(cID.Version)
	require.NoError(t, err)

	iter, _ := newStore.Iterator([]byte("aloha"), []byte("hellz"))
	expected := []string{"aloha", "hello"}
	var i int

	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}

	require.Equal(t, len(expected), i)
}

func TestIAVLStoreGetSetHasDelete(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := newAlohaTree(t, db)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)

	key := "hello"

	exists, _ := iavlStore.Has([]byte(key))
	require.True(t, exists)

	value, _ := iavlStore.Get([]byte(key))
	require.EqualValues(t, value, treeData[key])

	value2 := "notgoodbye"
	_ = iavlStore.Set([]byte(key), []byte(value2))

	value, _ = iavlStore.Get([]byte(key))
	require.EqualValues(t, value, value2)

	_ = iavlStore.Delete([]byte(key))

	exists, _ = iavlStore.Has([]byte(key))
	require.False(t, exists)
}

func TestIAVLStoreNoNilSet(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := newAlohaTree(t, db)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)
	require.Panics(t, func() { _ = iavlStore.Set([]byte("key"), nil) }, "setting a nil value should panic")
}

func TestIAVLIterator(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := newAlohaTree(t, db)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)
	iter, _ := iavlStore.Iterator([]byte("aloha"), []byte("hellz"))
	expected := []string{"aloha", "hello"}
	var i int

	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = iavlStore.Iterator([]byte("golang"), []byte("rocks"))
	expected = []string{"hello"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = iavlStore.Iterator(nil, []byte("golang"))
	expected = []string{"aloha"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = iavlStore.Iterator(nil, []byte("shalom"))
	expected = []string{"aloha", "hello"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = iavlStore.Iterator(nil, nil)
	expected = []string{"aloha", "hello"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = iavlStore.Iterator([]byte("golang"), nil)
	expected = []string{"hello"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, treeData[expectedKey])
		i++
	}
	require.Equal(t, len(expected), i)
}

func TestIAVLReverseIterator(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := NewMutableTree(db, cacheSize)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)

	_ = iavlStore.Set([]byte{0x00}, []byte("0"))
	_ = iavlStore.Set([]byte{0x00, 0x00}, []byte("0 0"))
	_ = iavlStore.Set([]byte{0x00, 0x01}, []byte("0 1"))
	_ = iavlStore.Set([]byte{0x00, 0x02}, []byte("0 2"))
	_ = iavlStore.Set([]byte{0x01}, []byte("1"))

	var testReverseIterator = func(t *testing.T, start []byte, end []byte, expected []string) {
		iter, _ := iavlStore.ReverseIterator(start, end)
		var i int
		for i = 0; iter.Valid(); iter.Next() {
			expectedValue := expected[i]
			value := iter.Value()
			require.EqualValues(t, string(value), expectedValue)
			i++
		}
		require.Equal(t, len(expected), i)
	}

	testReverseIterator(t, nil, nil, []string{"1", "0 2", "0 1", "0 0", "0"})
	testReverseIterator(t, []byte{0x00}, nil, []string{"1", "0 2", "0 1", "0 0", "0"})
	testReverseIterator(t, []byte{0x00}, []byte{0x00, 0x01}, []string{"0 0", "0"})
	testReverseIterator(t, []byte{0x00}, []byte{0x01}, []string{"0 2", "0 1", "0 0", "0"})
	testReverseIterator(t, []byte{0x00, 0x01}, []byte{0x01}, []string{"0 2", "0 1"})
	testReverseIterator(t, nil, []byte{0x01}, []string{"0 2", "0 1", "0 0", "0"})
}

func TestIAVLPrefixIterator(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := NewMutableTree(db, cacheSize)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)

	_ = iavlStore.Set([]byte("test1"), []byte("test1"))
	_ = iavlStore.Set([]byte("test2"), []byte("test2"))
	_ = iavlStore.Set([]byte("test3"), []byte("test3"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(0)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(1)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(255)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(0)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(1)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(255)}, []byte("test4"))

	var i int

	iter, _ := types.KVStorePrefixIterator(iavlStore, []byte("test"))
	expected := []string{"test1", "test2", "test3"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, expectedKey)
		i++
	}
	iter.Close()
	require.Equal(t, len(expected), i)

	iter, _ = types.KVStorePrefixIterator(iavlStore, []byte{byte(55), byte(255), byte(255)})
	expected2 := [][]byte{
		{byte(55), byte(255), byte(255), byte(0)},
		{byte(55), byte(255), byte(255), byte(1)},
		{byte(55), byte(255), byte(255), byte(255)},
	}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected2[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, []byte("test4"))
		i++
	}
	iter.Close()
	require.Equal(t, len(expected), i)

	iter, _ = types.KVStorePrefixIterator(iavlStore, []byte{byte(255), byte(255)})
	expected2 = [][]byte{
		{byte(255), byte(255), byte(0)},
		{byte(255), byte(255), byte(1)},
		{byte(255), byte(255), byte(255)},
	}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected2[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, []byte("test4"))
		i++
	}
	iter.Close()
	require.Equal(t, len(expected), i)
}

func TestIAVLReversePrefixIterator(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := NewMutableTree(db, cacheSize)
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)

	_ = iavlStore.Set([]byte("test1"), []byte("test1"))
	_ = iavlStore.Set([]byte("test2"), []byte("test2"))
	_ = iavlStore.Set([]byte("test3"), []byte("test3"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(0)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(1)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(55), byte(255), byte(255), byte(255)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(0)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(1)}, []byte("test4"))
	_ = iavlStore.Set([]byte{byte(255), byte(255), byte(255)}, []byte("test4"))

	var i int

	iter, _ := types.KVStoreReversePrefixIterator(iavlStore, []byte("test"))
	expected := []string{"test3", "test2", "test1"}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, expectedKey)
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = types.KVStoreReversePrefixIterator(iavlStore, []byte{byte(55), byte(255), byte(255)})
	expected2 := [][]byte{
		{byte(55), byte(255), byte(255), byte(255)},
		{byte(55), byte(255), byte(255), byte(1)},
		{byte(55), byte(255), byte(255), byte(0)},
	}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected2[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, []byte("test4"))
		i++
	}
	require.Equal(t, len(expected), i)

	iter, _ = types.KVStoreReversePrefixIterator(iavlStore, []byte{byte(255), byte(255)})
	expected2 = [][]byte{
		{byte(255), byte(255), byte(255)},
		{byte(255), byte(255), byte(1)},
		{byte(255), byte(255), byte(0)},
	}
	for i = 0; iter.Valid(); iter.Next() {
		expectedKey := expected2[i]
		key, value := iter.Key(), iter.Value()
		require.EqualValues(t, key, expectedKey)
		require.EqualValues(t, value, []byte("test4"))
		i++
	}
	require.Equal(t, len(expected), i)
}

func nextVersion(iavl *Store) {
	key := []byte(fmt.Sprintf("Key for tree: %d", iavl.LastCommitID().Version))
	value := []byte(fmt.Sprintf("Value for tree: %d", iavl.LastCommitID().Version))
	_ = iavl.Set(key, value)
	iavl.Commit()
}

func TestIAVLNoPrune(t *testing.T) {
	db := dbm.NewMemDB()
	tree, _ := NewMutableTree(db, cacheSize)
	iavlStore := UnsafeNewStore(tree, numRecent, int64(1))
	nextVersion(iavlStore)
	for i := 1; i < 100; i++ {
		for j := 1; j <= i; j++ {
			require.True(t, iavlStore.VersionExists(int64(j)),
				"Missing version %d with latest version %d. Should be storing all versions",
				j, i)
		}
		nextVersion(iavlStore)
	}
}

func BenchmarkIAVLIteratorNext(b *testing.B) {
	db := dbm.NewMemDB()
	treeSize := 1000
	tree, _ := NewMutableTree(db, cacheSize)
	for i := 0; i < treeSize; i++ {
		key := rand2.Bytes(4)
		value := rand2.Bytes(50)
		tree.Set(key, value)
	}
	iavlStore := UnsafeNewStore(tree, numRecent, storeEvery)
	iterators := make([]types.Iterator, b.N/treeSize)
	for i := 0; i < len(iterators); i++ {
		iterators[i], _ = iavlStore.Iterator([]byte{0}, []byte{255, 255, 255, 255, 255})
	}
	b.ResetTimer()
	for i := 0; i < len(iterators); i++ {
		iter := iterators[i]
		for j := 0; j < treeSize; j++ {
			iter.Next()
		}
	}
}
