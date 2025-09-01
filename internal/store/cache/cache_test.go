package cache_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/stretchr/testify/require"
)

func TestXSync(t *testing.T) {
	c := xsync.NewMap[string, string]()

	// store data
	c.Store("key", "value")

	// load data
	value, ok := c.Load("key")
	require.Equal(t, true, ok)
	require.Equal(t, "value", value)

	// remove data
	c.Delete("key")

	// load data
	value, ok = c.Load("key")
	require.Equal(t, false, ok)
	require.Equal(t, "", value)

	// store data
	c.Store("key", "value2")

	// load data
	value, ok = c.Load("key")
	require.Equal(t, true, ok)
	require.Equal(t, "value2", value)

	// clear
	c.Clear()

	// load data
	value, ok = c.Load("key")
	require.Equal(t, false, ok)
	require.Equal(t, "", value)
}

func TestXSync2(t *testing.T) {
	c := xsync.NewMap[string, struct{}]()

	keysCount := 500000
	workersCount := 100

	for i := range keysCount {
		c.Store(fmt.Sprintf("key_%d", i), struct{}{})
	}

	wg := sync.WaitGroup{}

	for i := range workersCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Store(fmt.Sprintf("key_%d", i), struct{}{})
		}()
	}

	t.Logf("keys count: %d", c.Size())

	keyNotFound := atomic.Bool{}
	for i := range workersCount {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Range(func(key string, _ struct{}) bool {
				return true
			})
		}()
		go func(i int) {
			defer wg.Done()
			for i := range keysCount {
				_, ok := c.Load(fmt.Sprintf("key_%d", i))
				if !ok {
					keyNotFound.Store(true)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	require.Equal(t, false, keyNotFound.Load())
}
