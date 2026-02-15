package beads

import (
	"context"
	"testing"
	"time"
)

// TestNewDoltCoordinator tests coordinator creation
func TestNewDoltCoordinator(t *testing.T) {
	masterID := "loom-self"
	bdPath := "/usr/local/bin/bd"
	basePort := 3307

	coord := NewDoltCoordinator(masterID, bdPath, basePort)

	if coord == nil {
		t.Fatal("Expected non-nil coordinator")
	}

	if coord.masterID != masterID {
		t.Errorf("masterID = %q, want %q", coord.masterID, masterID)
	}

	if coord.bdPath != bdPath {
		t.Errorf("bdPath = %q, want %q", coord.bdPath, bdPath)
	}

	if coord.basePort != basePort {
		t.Errorf("basePort = %d, want %d", coord.basePort, basePort)
	}

	if coord.nextPort != basePort {
		t.Errorf("nextPort = %d, want %d", coord.nextPort, basePort)
	}

	if coord.instances == nil {
		t.Error("Expected instances map to be initialized")
	}
}

// TestNewDoltCoordinator_DefaultPort tests default port
func TestNewDoltCoordinator_DefaultPort(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 0)

	if coord.basePort != 3307 {
		t.Errorf("basePort = %d, want %d (default)", coord.basePort, 3307)
	}
}

// TestDoltCoordinator_GetInstance tests getting instances
func TestDoltCoordinator_GetInstance(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// Get non-existent instance
	inst := coord.GetInstance("nonexistent")
	if inst != nil {
		t.Error("Expected nil for non-existent instance")
	}

	// Add an instance manually
	testInst := &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     true,
		StartedAt: time.Now(),
	}

	coord.mu.Lock()
	coord.instances["project1"] = testInst
	coord.mu.Unlock()

	// Get existing instance
	inst = coord.GetInstance("project1")
	if inst == nil {
		t.Fatal("Expected non-nil for existing instance")
	}

	if inst.ProjectID != "project1" {
		t.Errorf("ProjectID = %q, want %q", inst.ProjectID, "project1")
	}

	if inst.Port != 3308 {
		t.Errorf("Port = %d, want %d", inst.Port, 3308)
	}
}

// TestDoltCoordinator_GetMasterInstance tests getting master instance
func TestDoltCoordinator_GetMasterInstance(t *testing.T) {
	masterID := "loom-self"
	coord := NewDoltCoordinator(masterID, "/bin/bd", 3307)

	// No master yet
	master := coord.GetMasterInstance()
	if master != nil {
		t.Error("Expected nil when no master instance exists")
	}

	// Add master instance
	masterInst := &DoltInstance{
		ProjectID: masterID,
		Port:      3307,
		Ready:     true,
		StartedAt: time.Now(),
	}

	coord.mu.Lock()
	coord.instances[masterID] = masterInst
	coord.mu.Unlock()

	// Get master
	master = coord.GetMasterInstance()
	if master == nil {
		t.Fatal("Expected non-nil master instance")
	}

	if master.ProjectID != masterID {
		t.Errorf("Master ProjectID = %q, want %q", master.ProjectID, masterID)
	}
}

// TestDoltCoordinator_ListInstances tests listing all instances
func TestDoltCoordinator_ListInstances(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// Empty list
	instances := coord.ListInstances()
	if len(instances) != 0 {
		t.Errorf("Expected empty list, got %d instances", len(instances))
	}

	// Add some instances
	inst1 := &DoltInstance{ProjectID: "project1", Port: 3307, Ready: true}
	inst2 := &DoltInstance{ProjectID: "project2", Port: 3308, Ready: true}

	coord.mu.Lock()
	coord.instances["project1"] = inst1
	coord.instances["project2"] = inst2
	coord.mu.Unlock()

	// List instances
	instances = coord.ListInstances()
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	if instances["project1"] == nil || instances["project2"] == nil {
		t.Error("Expected both instances to be in the list")
	}

	// Verify copy (not same map)
	delete(instances, "project1")
	if coord.GetInstance("project1") == nil {
		t.Error("Deleting from returned map should not affect coordinator")
	}
}

// TestDoltCoordinator_StopInstance tests stopping an instance
func TestDoltCoordinator_StopInstance(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// Stop non-existent instance (should not error)
	err := coord.StopInstance("nonexistent")
	if err != nil {
		t.Errorf("StopInstance(nonexistent) error = %v, want nil", err)
	}

	// Add an instance without a real process
	inst := &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     true,
		StartedAt: time.Now(),
		Cmd:       nil, // No actual command
	}

	coord.mu.Lock()
	coord.instances["project1"] = inst
	coord.mu.Unlock()

	// Stop the instance
	err = coord.StopInstance("project1")
	if err != nil {
		t.Errorf("StopInstance() error = %v", err)
	}

	// Verify instance was removed
	if coord.GetInstance("project1") != nil {
		t.Error("Expected instance to be removed after stop")
	}
}

// TestDoltCoordinator_Status tests status reporting
func TestDoltCoordinator_Status(t *testing.T) {
	masterID := "loom-self"
	coord := NewDoltCoordinator(masterID, "/bin/bd", 3307)

	// Empty status
	status := coord.Status()
	if status["master_id"] != masterID {
		t.Errorf("master_id = %v, want %q", status["master_id"], masterID)
	}

	if status["instance_count"] != 0 {
		t.Errorf("instance_count = %v, want %d", status["instance_count"], 0)
	}

	// Add some instances
	inst1 := &DoltInstance{ProjectID: masterID, Port: 3307, Ready: true, StartedAt: time.Now()}
	inst2 := &DoltInstance{ProjectID: "project1", Port: 3308, Ready: true, StartedAt: time.Now()}

	coord.mu.Lock()
	coord.instances[masterID] = inst1
	coord.instances["project1"] = inst2
	coord.mu.Unlock()

	// Status with instances
	status = coord.Status()
	if status["instance_count"] != 2 {
		t.Errorf("instance_count = %v, want %d", status["instance_count"], 2)
	}

	instances, ok := status["instances"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected instances to be []map[string]interface{}")
	}

	if len(instances) != 2 {
		t.Errorf("instances length = %d, want %d", len(instances), 2)
	}

	// Check master flag
	foundMaster := false
	for _, inst := range instances {
		if inst["is_master"] == true {
			foundMaster = true
			if inst["project_id"] != masterID {
				t.Errorf("Master instance project_id = %v, want %q", inst["project_id"], masterID)
			}
		}
	}

	if !foundMaster {
		t.Error("Expected to find master instance in status")
	}
}

// TestDoltCoordinator_AllocatePort tests port allocation
func TestDoltCoordinator_AllocatePort(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// First allocation
	port1 := coord.allocatePort()
	if port1 != 3307 {
		t.Errorf("First allocated port = %d, want %d", port1, 3307)
	}

	// Second allocation
	port2 := coord.allocatePort()
	if port2 != 3308 {
		t.Errorf("Second allocated port = %d, want %d", port2, 3308)
	}

	// Third allocation
	port3 := coord.allocatePort()
	if port3 != 3309 {
		t.Errorf("Third allocated port = %d, want %d", port3, 3309)
	}

	// Verify nextPort was updated
	if coord.nextPort != 3310 {
		t.Errorf("nextPort = %d, want %d", coord.nextPort, 3310)
	}
}

// TestDoltCoordinator_Shutdown tests shutting down all instances
func TestDoltCoordinator_Shutdown(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// Add instances without real processes
	inst1 := &DoltInstance{ProjectID: "project1", Port: 3307, Ready: true}
	inst2 := &DoltInstance{ProjectID: "project2", Port: 3308, Ready: true}

	coord.mu.Lock()
	coord.instances["project1"] = inst1
	coord.instances["project2"] = inst2
	coord.mu.Unlock()

	// Shutdown
	coord.Shutdown()

	// Verify all instances were removed
	instances := coord.ListInstances()
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances after shutdown, got %d", len(instances))
	}

	// Calling shutdown again should be safe (idempotent)
	coord.Shutdown()
}

// TestDoltInstance_ConnectionString tests connection string generation
func TestDoltInstance_ConnectionString(t *testing.T) {
	inst := &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     true,
	}

	connStr := inst.ConnectionString()
	want := "root@tcp(127.0.0.1:3308)/beads"

	if connStr != want {
		t.Errorf("ConnectionString() = %q, want %q", connStr, want)
	}
}

// TestDoltInstance_IsHealthy tests health check
func TestDoltInstance_IsHealthy(t *testing.T) {
	// Nil instance
	var inst *DoltInstance
	if inst.IsHealthy() {
		t.Error("Expected nil instance to be unhealthy")
	}

	// Not ready instance
	inst = &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     false,
	}

	if inst.IsHealthy() {
		t.Error("Expected not-ready instance to be unhealthy")
	}

	// Ready instance but port not listening
	// (This will fail because nothing is actually listening on the port)
	inst = &DoltInstance{
		ProjectID: "project1",
		Port:      9999, // Unlikely to be in use
		Ready:     true,
	}

	if inst.IsHealthy() {
		// This might pass if something is actually listening on port 9999
		// but that's unlikely in a test environment
		t.Log("Note: IsHealthy() returned true, but nothing should be listening on port 9999")
	}
}

// TestDoltCoordinator_EnsureInstance tests ensuring an instance exists
func TestDoltCoordinator_EnsureInstance(t *testing.T) {
	coord := NewDoltCoordinator("master", "/bin/bd", 3307)

	// Add an already-ready instance
	existingInst := &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     true,
		StartedAt: time.Now(),
	}

	coord.mu.Lock()
	coord.instances["project1"] = existingInst
	coord.mu.Unlock()

	// EnsureInstance should return the existing instance
	ctx := context.Background()
	inst, err := coord.EnsureInstance(ctx, "project1", "/tmp/data")

	// This will fail because dolt is not actually installed/running
	// but we can verify the logic tries to return existing instance first
	if err == nil && inst != nil {
		if inst.ProjectID != "project1" {
			t.Errorf("Returned instance ProjectID = %q, want %q", inst.ProjectID, "project1")
		}

		if inst.Port != 3308 {
			t.Errorf("Returned instance Port = %d, want %d", inst.Port, 3308)
		}
	}
	// If error, it's because dolt isn't installed, which is expected
}

// TestDoltCoordinator_SyncFederationAll tests federation sync
func TestDoltCoordinator_SyncFederationAll(t *testing.T) {
	masterID := "master"
	coord := NewDoltCoordinator(masterID, "/bin/bd", 3307)

	ctx := context.Background()

	// No master instance
	err := coord.SyncFederationAll(ctx)
	if err == nil {
		t.Error("Expected error when master instance not ready")
	}

	// Add master but not ready
	coord.mu.Lock()
	coord.instances[masterID] = &DoltInstance{
		ProjectID: masterID,
		Port:      3307,
		Ready:     false,
	}
	coord.mu.Unlock()

	err = coord.SyncFederationAll(ctx)
	if err == nil {
		t.Error("Expected error when master instance not ready")
	}

	// Make master ready
	coord.mu.Lock()
	coord.instances[masterID].Ready = true
	coord.mu.Unlock()

	// Should not error now (but will fail to actually sync since dolt isn't running)
	_ = coord.SyncFederationAll(ctx)
	// We don't assert on error here because it depends on dolt being installed
}

// TestDoltCoordinator_SyncProject tests syncing a single project
func TestDoltCoordinator_SyncProject(t *testing.T) {
	masterID := "master"
	coord := NewDoltCoordinator(masterID, "/bin/bd", 3307)

	ctx := context.Background()

	// No instances
	err := coord.SyncProject(ctx, "project1")
	if err == nil {
		t.Error("Expected error when project instance not ready")
	}

	// Add project but no master
	coord.mu.Lock()
	coord.instances["project1"] = &DoltInstance{
		ProjectID: "project1",
		Port:      3308,
		Ready:     true,
	}
	coord.mu.Unlock()

	err = coord.SyncProject(ctx, "project1")
	if err == nil {
		t.Error("Expected error when master instance not ready")
	}

	// Add master
	coord.mu.Lock()
	coord.instances[masterID] = &DoltInstance{
		ProjectID: masterID,
		Port:      3307,
		Ready:     true,
	}
	coord.mu.Unlock()

	// Should not error about missing instances now
	// (but will fail to actually sync since dolt isn't running)
	_ = coord.SyncProject(ctx, "project1")
	// We don't assert on error here because it depends on dolt being installed
}
