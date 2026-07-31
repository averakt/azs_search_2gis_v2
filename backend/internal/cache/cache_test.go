package cache

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestGetSet(t *testing.T) {
	c := New(10 * time.Minute)

	_, found := c.Get("missing")
	if found {
		t.Error("Get() for missing key should return false")
	}

	c.Set("key1", "value1")
	v, found := c.Get("key1")
	if !found {
		t.Fatal("Get() for existing key should return true")
	}
	if v.(string) != "value1" {
		t.Errorf("Get() = %v, want %v", v, "value1")
	}

	c.Set("key2", 42)
	v, found = c.Get("key2")
	if !found {
		t.Fatal("Get() for existing key should return true")
	}
	if v.(int) != 42 {
		t.Errorf("Get() = %v, want %v", v, 42)
	}
}

func TestGetOrLoadHit(t *testing.T) {
	c := New(10 * time.Minute)
	c.Set("existing", "cached_value")

	loadCount := 0
	v, err := c.GetOrLoad("existing", func() (interface{}, error) {
		loadCount++
		return "should_not_be_called", nil
	})

	if err != nil {
		t.Fatalf("GetOrLoad() error = %v", err)
	}
	if v.(string) != "cached_value" {
		t.Errorf("GetOrLoad() = %v, want %v", v, "cached_value")
	}
	if loadCount != 0 {
		t.Error("load function should not be called on cache hit")
	}
}

func TestGetOrLoadMiss(t *testing.T) {
	c := New(10 * time.Minute)

	loadCount := 0
	v, err := c.GetOrLoad("missing", func() (interface{}, error) {
		loadCount++
		return "loaded_value", nil
	})

	if err != nil {
		t.Fatalf("GetOrLoad() error = %v", err)
	}
	if v.(string) != "loaded_value" {
		t.Errorf("GetOrLoad() = %v, want %v", v, "loaded_value")
	}
	if loadCount != 1 {
		t.Errorf("load function called %d times, want 1", loadCount)
	}

	v, err = c.GetOrLoad("missing", func() (interface{}, error) {
		loadCount++
		return "should_not_be_called", nil
	})
	if err != nil {
		t.Fatalf("GetOrLoad() error = %v", err)
	}
	if v.(string) != "loaded_value" {
		t.Errorf("GetOrLoad() after miss = %v, want %v", v, "loaded_value")
	}
	if loadCount != 1 {
		t.Errorf("load function should not be called again, called %d times", loadCount)
	}
}

func TestGetOrLoadError(t *testing.T) {
	c := New(10 * time.Minute)

	loadErr := errors.New("load error")
	_, err := c.GetOrLoad("error_key", func() (interface{}, error) {
		return nil, loadErr
	})

	if !errors.Is(err, loadErr) {
		t.Errorf("GetOrLoad() error = %v, want %v", err, loadErr)
	}

	_, found := c.Get("error_key")
	if found {
		t.Error("errored load should not cache the result")
	}
}

func TestGetOrLoadConcurrent(t *testing.T) {
	c := New(10 * time.Minute)

	var mu sync.Mutex
	loadCount := 0

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetOrLoad("concurrent", func() (interface{}, error) {
				mu.Lock()
				loadCount++
				mu.Unlock()
				return "result", nil
			})
			if err != nil {
				t.Errorf("GetOrLoad() error = %v", err)
			}
			if v.(string) != "result" {
				t.Errorf("GetOrLoad() = %v, want %v", v, "result")
			}
		}()
	}
	wg.Wait()

	if loadCount != 1 {
		t.Errorf("load function called %d times, want 1 (singleflight)", loadCount)
	}
}

func TestDelete(t *testing.T) {
	c := New(10 * time.Minute)

	c.Set("todelete", "value")
	_, found := c.Get("todelete")
	if !found {
		t.Fatal("value should exist before delete")
	}

	c.Delete("todelete")
	_, found = c.Get("todelete")
	if found {
		t.Error("value should not exist after delete")
	}

	c.Delete("nonexistent")
}
