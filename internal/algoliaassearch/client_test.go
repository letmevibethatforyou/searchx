package algoliaassearch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestStaticSecrets(t *testing.T) {
	appID := "test-app-id"
	apiKey := "test-api-key"

	fetchSecrets := StaticSecrets(appID, apiKey)
	secrets, err := fetchSecrets()

	if err != nil {
		t.Errorf("StaticSecrets should not return error, got: %v", err)
	}

	if secrets.AppID != appID {
		t.Errorf("Expected AppID %s, got %s", appID, secrets.AppID)
	}

	if secrets.WriteApiKey != apiKey {
		t.Errorf("Expected WriteApiKey %s, got %s", apiKey, secrets.WriteApiKey)
	}
}

func TestEnvSecrets(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid secrets",
			appID:       "test-app-id",
			apiKey:      "test-api-key",
			expectError: false,
		},
		{
			name:        "missing app id",
			appID:       "",
			apiKey:      "test-api-key",
			expectError: true,
		},
		{
			name:        "missing api key",
			appID:       "test-app-id",
			apiKey:      "",
			expectError: true,
		},
		{
			name:        "both missing",
			appID:       "",
			apiKey:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			t.Setenv("ALGOLIA_APP_ID", tt.appID)
			t.Setenv("ALGOLIA_API_KEY", tt.apiKey)

			fetchSecrets := EnvSecrets()
			secrets, err := fetchSecrets()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if secrets.AppID != tt.appID {
					t.Errorf("Expected AppID %s, got %s", tt.appID, secrets.AppID)
				}
				if secrets.WriteApiKey != tt.apiKey {
					t.Errorf("Expected WriteApiKey %s, got %s", tt.apiKey, secrets.WriteApiKey)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name            string
		fetchSecrets    FetchSecrets
		expectInitError bool
	}{
		{
			name:            "valid secrets",
			fetchSecrets:    StaticSecrets("test-app", "test-key"),
			expectInitError: false,
		},
		{
			name: "fetch error",
			fetchSecrets: func() (Secrets, error) {
				return Secrets{}, errors.New("fetch failed")
			},
			expectInitError: true,
		},
		{
			name:            "empty app id",
			fetchSecrets:    StaticSecrets("", "test-key"),
			expectInitError: true,
		},
		{
			name:            "empty api key",
			fetchSecrets:    StaticSecrets("test-app", ""),
			expectInitError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.fetchSecrets)

			if client == nil {
				t.Error("NewClient should never return nil")
			}

			// Test that the client initialization works by calling a method
			ctx := context.Background()
			err := client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})

			if tt.expectInitError {
				if err == nil {
					t.Error("Expected initialization error but got none")
				}
			}
			// For valid credentials, we may get Algolia API errors, which is fine
		})
	}
}

func TestClientLazyInitialization(t *testing.T) {
	callCount := 0
	fetchSecrets := func() (Secrets, error) {
		callCount++
		return Secrets{
			AppID:       "test-app",
			WriteApiKey: "test-key",
		}, nil
	}

	client := NewClient(fetchSecrets)

	// Constructor should not call fetchSecrets
	if callCount != 0 {
		t.Errorf("Expected fetchSecrets not to be called during construction, but was called %d times", callCount)
	}

	ctx := context.Background()

	// First method call should trigger fetchSecrets
	_ = client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})
	if callCount != 1 {
		t.Errorf("Expected fetchSecrets to be called once after first method call, but was called %d times", callCount)
	}

	// Second method call should not call fetchSecrets again
	_ = client.DeleteObject(ctx, "test-index", "test-id")
	if callCount != 1 {
		t.Errorf("Expected fetchSecrets to still be called only once after second method call, but was called %d times", callCount)
	}

	// Third method call should not call fetchSecrets again
	_ = client.BatchSaveObjects(ctx, "test-index", []map[string]interface{}{{"test": "data"}})
	if callCount != 1 {
		t.Errorf("Expected fetchSecrets to still be called only once after third method call, but was called %d times", callCount)
	}
}

func TestClientErrorCaching(t *testing.T) {
	callCount := 0
	fetchSecrets := func() (Secrets, error) {
		callCount++
		return Secrets{}, errors.New("simulated fetch error")
	}

	client := NewClient(fetchSecrets)
	ctx := context.Background()

	// First call should trigger fetchSecrets and return error
	err1 := client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})
	if err1 == nil {
		t.Error("Expected error from first call")
	}
	if callCount != 1 {
		t.Errorf("Expected fetchSecrets to be called once, but was called %d times", callCount)
	}

	// Second call should return cached error without calling fetchSecrets again
	err2 := client.BatchSaveObjects(ctx, "test-index", []map[string]interface{}{{"test": "data"}})
	if err2 == nil {
		t.Error("Expected error from second call")
	}
	if callCount != 1 {
		t.Errorf("Expected fetchSecrets to still be called only once, but was called %d times", callCount)
	}

	// Both errors should be the same (cached)
	if err1.Error() != err2.Error() {
		t.Errorf("Expected same error messages, got '%s' and '%s'", err1.Error(), err2.Error())
	}
}

func TestClientConcurrentAccess(t *testing.T) {
	var callCount int32
	var mu sync.Mutex
	fetchSecrets := func() (Secrets, error) {
		mu.Lock()
		callCount++
		mu.Unlock()

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		return Secrets{
			AppID:       "test-app",
			WriteApiKey: "test-key",
		}, nil
	}

	client := NewClient(fetchSecrets)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	wg.Add(numGoroutines)

	// Launch multiple goroutines that call client methods concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_ = client.SaveObject(ctx, "test-index", map[string]interface{}{"goroutine": id})
		}(i)
	}

	wg.Wait()

	mu.Lock()
	finalCallCount := callCount
	mu.Unlock()

	if finalCallCount != 1 {
		t.Errorf("Expected fetchSecrets to be called exactly once despite concurrent access, but was called %d times", finalCallCount)
	}
}

func TestClientMethods(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	ctx := context.Background()

	t.Run("SaveObject", func(t *testing.T) {
		_ = client.SaveObject(ctx, "test-index", map[string]interface{}{
			"name":  "test object",
			"value": 42,
		})
		// Method should not panic, error handling is tested elsewhere
	})

	t.Run("DeleteObject", func(t *testing.T) {
		_ = client.DeleteObject(ctx, "test-index", "test-object-id")
		// Method should not panic, error handling is tested elsewhere
	})

	t.Run("BatchSaveObjects with data", func(t *testing.T) {
		objects := []map[string]interface{}{
			{"id": 1, "name": "object1"},
			{"id": 2, "name": "object2"},
		}
		_ = client.BatchSaveObjects(ctx, "test-index", objects)
		// Method should not panic, error handling is tested elsewhere
	})

	t.Run("BatchSaveObjects empty slice", func(t *testing.T) {
		err := client.BatchSaveObjects(ctx, "test-index", []map[string]interface{}{})
		if err != nil {
			t.Errorf("BatchSaveObjects with empty slice should return nil, got: %v", err)
		}
	})

	t.Run("BatchDeleteObjects with data", func(t *testing.T) {
		objectIDs := []string{"id1", "id2", "id3"}
		_ = client.BatchDeleteObjects(ctx, "test-index", objectIDs)
		// Method should not panic, error handling is tested elsewhere
	})

	t.Run("BatchDeleteObjects empty slice", func(t *testing.T) {
		err := client.BatchDeleteObjects(ctx, "test-index", []string{})
		if err != nil {
			t.Errorf("BatchDeleteObjects with empty slice should return nil, got: %v", err)
		}
	})
}

func TestClientMethodsWithInitializationErrors(t *testing.T) {
	tests := []struct {
		name         string
		fetchSecrets FetchSecrets
	}{
		{
			name: "fetch error",
			fetchSecrets: func() (Secrets, error) {
				return Secrets{}, errors.New("fetch failed")
			},
		},
		{
			name:         "empty app id",
			fetchSecrets: StaticSecrets("", "test-key"),
		},
		{
			name:         "empty api key",
			fetchSecrets: StaticSecrets("test-app", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.fetchSecrets)
			ctx := context.Background()

			// Test SaveObject
			err := client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})
			if err == nil {
				t.Error("Expected SaveObject to return initialization error")
			}

			// Test DeleteObject
			err = client.DeleteObject(ctx, "test-index", "test-id")
			if err == nil {
				t.Error("Expected DeleteObject to return initialization error")
			}

			// Test BatchSaveObjects
			err = client.BatchSaveObjects(ctx, "test-index", []map[string]interface{}{{"test": "data"}})
			if err == nil {
				t.Error("Expected BatchSaveObjects to return initialization error")
			}

			// Test BatchDeleteObjects
			err = client.BatchDeleteObjects(ctx, "test-index", []string{"test-id"})
			if err == nil {
				t.Error("Expected BatchDeleteObjects to return initialization error")
			}
		})
	}
}

func TestClientNilInputHandling(t *testing.T) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	ctx := context.Background()

	t.Run("SaveObject with nil object", func(t *testing.T) {
		// This should not panic
		_ = client.SaveObject(ctx, "test-index", nil)
	})

	t.Run("BatchSaveObjects with nil slice", func(t *testing.T) {
		err := client.BatchSaveObjects(ctx, "test-index", nil)
		// nil slice should be treated as empty and return no error
		if err != nil {
			t.Errorf("BatchSaveObjects with nil slice should return no error, got: %v", err)
		}
	})

	t.Run("BatchDeleteObjects with nil slice", func(t *testing.T) {
		err := client.BatchDeleteObjects(ctx, "test-index", nil)
		// nil slice should be treated as empty and return no error
		if err != nil {
			t.Errorf("BatchDeleteObjects with nil slice should return no error, got: %v", err)
		}
	})
}

// Benchmark tests to ensure performance characteristics
func BenchmarkClientInitialization(b *testing.B) {
	fetchSecrets := StaticSecrets("test-app", "test-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := NewClient(fetchSecrets)
		ctx := context.Background()

		// This should trigger initialization only once per client
		_ = client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})
	}
}

func BenchmarkClientReuse(b *testing.B) {
	client := NewClient(StaticSecrets("test-app", "test-key"))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// These should reuse the initialized client
		_ = client.SaveObject(ctx, "test-index", map[string]interface{}{"test": "data"})
	}
}
