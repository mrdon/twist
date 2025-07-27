//go:build integration

package scripting

import (
	"testing"
	"twist/internal/scripting/manager"
)

// TestScriptManager_LoadScript_RealIntegration tests script loading with real database
func TestScriptManager_LoadScript_RealIntegration(t *testing.T) {
	setup := NewIntegrationScriptTester(t)
	sm := manager.NewScriptManager(setup.setupData.DB)
	
	tests := []struct {
		name     string
		filename string
		isSystem bool
		wantErr  bool
	}{
		{
			name:     "Load user script",
			filename: "test_script.twx",
			isSystem: false,
			wantErr:  false,
		},
		{
			name:     "Load system script",
			filename: "system/backup.twx",
			isSystem: true,
			wantErr:  false,
		},
		{
			name:     "Load script with path",
			filename: "scripts/trading/trader.twx",
			isSystem: false,
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptInfo, err := sm.LoadScript(tt.filename, tt.isSystem)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadScript() expected error, got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("LoadScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if scriptInfo == nil {
				t.Error("LoadScript() returned nil ScriptInfo")
				return
			}
			
			// Verify script info
			if scriptInfo.Filename != tt.filename {
				t.Errorf("ScriptInfo.Filename = %v, expected %v", scriptInfo.Filename, tt.filename)
			}
			
			if scriptInfo.System != tt.isSystem {
				t.Errorf("ScriptInfo.System = %v, expected %v", scriptInfo.System, tt.isSystem)
			}
			
			if scriptInfo.Running {
				t.Error("ScriptInfo.Running should be false for newly loaded script")
			}
			
			if scriptInfo.ID == "" {
				t.Error("ScriptInfo.ID should not be empty")
			}
		})
	}
}

// TestScriptManager_StopScript_RealIntegration tests script stopping with real database
func TestScriptManager_StopScript_RealIntegration(t *testing.T) {
	setup := NewIntegrationScriptTester(t)
	sm := manager.NewScriptManager(setup.setupData.DB)
	
	// Load a script first
	scriptInfo, err := sm.LoadScript("test_stop.twx", false)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}
	
	// Verify script exists in manager
	info, exists := sm.GetScriptByID(scriptInfo.ID)
	if !exists {
		t.Fatalf("Script not found by ID: %v", scriptInfo.ID)
	}
	
	if info.ID != scriptInfo.ID {
		t.Errorf("Retrieved script ID = %v, expected %v", info.ID, scriptInfo.ID)
	}
	
	// Stop the script
	err = sm.StopScript(scriptInfo.ID)
	if err != nil {
		t.Errorf("StopScript() error = %v", err)
	}
	
	// Verify script still exists but can be retrieved
	stoppedInfo, exists := sm.GetScriptByID(scriptInfo.ID)
	if !exists {
		t.Error("Script should still exist after StopScript")
	} else if stoppedInfo.StoppedAt == nil {
		t.Error("Script StoppedAt should be set after StopScript")
	}
}

// TestScriptManager_ListScripts_RealIntegration tests script listing with real database
func TestScriptManager_ListScripts_RealIntegration(t *testing.T) {
	setup := NewIntegrationScriptTester(t)
	sm := manager.NewScriptManager(setup.setupData.DB)
	
	// Load multiple scripts
	scripts := []struct {
		filename string
		isSystem bool
	}{
		{"user_script1.twx", false},
		{"user_script2.twx", false},
		{"system/admin.twx", true},
		{"system/backup.twx", true},
	}
	
	loadedIDs := make([]string, len(scripts))
	for i, script := range scripts {
		scriptInfo, err := sm.LoadScript(script.filename, script.isSystem)
		if err != nil {
			t.Fatalf("Failed to load script %s: %v", script.filename, err)
		}
		loadedIDs[i] = scriptInfo.ID
	}
	
	// Get active scripts (should include our loaded scripts)
	activeScripts := sm.GetActiveScripts()
	if len(activeScripts) < len(scripts) {
		t.Errorf("GetActiveScripts() returned %d scripts, expected at least %d", len(activeScripts), len(scripts))
	}
	
	// Verify all loaded scripts can be retrieved by ID
	for i, expectedID := range loadedIDs {
		script, exists := sm.GetScriptByID(expectedID)
		if !exists {
			t.Errorf("Script %d (ID: %s) not found by GetScriptByID", i, expectedID)
		} else {
			if script.ID != expectedID {
				t.Errorf("Script %d ID mismatch: got %s, expected %s", i, script.ID, expectedID)
			}
		}
	}
	
	// Test retrieval by name
	for i, scriptDef := range scripts {
		script, exists := sm.GetScriptByName(scriptDef.filename)
		if !exists {
			t.Errorf("Script %d (name: %s) not found by GetScriptByName", i, scriptDef.filename)
		} else {
			if script.Filename != scriptDef.filename {
				t.Errorf("Script %d filename mismatch: got %s, expected %s", i, script.Filename, scriptDef.filename)
			}
		}
	}
}

// TestScriptManager_DatabasePersistence_RealIntegration tests persistence across instances
func TestScriptManager_DatabasePersistence_RealIntegration(t *testing.T) {
	// First VM instance
	setup1 := NewIntegrationScriptTester(t)
	sm1 := manager.NewScriptManager(setup1.setupData.DB)
	
	// Load a script
	scriptInfo, err := sm1.LoadScript("persistent_test.twx", false)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}
	
	originalID := scriptInfo.ID
	
	// Second VM instance with shared database
	setup2 := NewIntegrationScriptTesterWithSharedDB(t, setup1.setupData)
	sm2 := manager.NewScriptManager(setup2.setupData.DB)
	
	// Restore scripts from database
	err = sm2.RestoreFromDatabase()
	if err != nil {
		t.Fatalf("Failed to restore from database: %v", err)
	}
	
	// Verify script was restored by ID
	restoredInfo, exists := sm2.GetScriptByID(originalID)
	if !exists {
		t.Fatalf("Script not found after restoration: %v", originalID)
	}
	
	if restoredInfo.ID != originalID {
		t.Errorf("Restored script ID = %v, expected %v", restoredInfo.ID, originalID)
	}
	
	if restoredInfo.Filename != "persistent_test.twx" {
		t.Errorf("Restored script filename = %v, expected persistent_test.twx", restoredInfo.Filename)
	}
	
	// Verify we can control the restored script
	err = sm2.StopScript(originalID)
	if err != nil {
		t.Errorf("Failed to stop restored script: %v", err)
	}
	
	// Verify script was stopped by checking it still exists but is stopped
	stoppedInfo, exists := sm2.GetScriptByID(originalID)
	if !exists {
		t.Error("Script should still exist after stopping")
	} else if stoppedInfo.StoppedAt == nil {
		t.Error("Script should have StoppedAt timestamp after StopScript call")
	}
}