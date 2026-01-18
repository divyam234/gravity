package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestConstants
const (
	ServerBinPath = "../../cmd/server/server"
	ServerPort    = 8080
	DBPath        = "test_data/aria2-rclone.db"
	Aria2Secret   = "test-secret"
)

func TestAria2BoltDBIntegration(t *testing.T) {
	// 1. Setup
	// Compile Server
	cmdBuild := exec.Command("go", "build", "-o", "server_test", "../../cmd/server/main.go")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("server_test")

	// Prepare Test Data Dir
	os.RemoveAll("test_data")
	os.MkdirAll("test_data", 0755)

	// Override DB Path via ENV
	// We pass environment to the command instead of setting it globally
	// to avoid pollution.
	wd, _ := os.Getwd()
	mockHome := filepath.Join(wd, "test_data")

	serverEnv := os.Environ()
	serverEnv = append(serverEnv, fmt.Sprintf("HOME=%s", mockHome))
	serverEnv = append(serverEnv, fmt.Sprintf("ARIA2_SECRET=%s", Aria2Secret))

	// 2. Start Server
	serverCmd := exec.Command("./server_test")
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	serverCmd.Env = serverEnv

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// Wait for startup
	time.Sleep(5 * time.Second)

	// 3. Test: Add Download (URI)
	// We send RPC to our proxy (port 8080)
	rpcUrl := fmt.Sprintf("http://localhost:%d/jsonrpc", ServerPort)

	// Payload for aria2.addUri
	addUriParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.addUri",
		"id":      "test-1",
		"params": []interface{}{
			"token:" + Aria2Secret,
			[]string{"https://mmatechnical.com/Download/Download-Test-File/(MMA)-100MB.zip"},
			map[string]string{
				"rclone-target": "my-remote:", // Our custom option
				"dir":           filepath.Join(mockHome, "Downloads"),
			},
		},
	}

	resp := sendRPC(t, rpcUrl, addUriParams)
	if resp.Error != nil {
		t.Fatalf("RPC Error: %v", resp.Error)
	}
	gid := resp.Result.(string)
	t.Logf("Added Task GID: %s", gid)

	// 4. Verify BoltDB Persistence
	// We need to open the DB and check if GID exists.
	// Since server has lock, we might not be able to open it unless we use ReadOnly mode or kill server.
	// But bbolt supports multiple readers?
	// Let's verify via Side-Channel: Restart Server and see if task resumes?
	// Or query our own API?
	// Currently we don't have an API to "List DB contents" explicitly except via tellStatus if active.

	// Let's check tellStatus via Proxy
	statusParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.tellStatus",
		"id":      "test-2",
		"params": []interface{}{
			"token:" + Aria2Secret,
			gid,
		},
	}
	statusResp := sendRPC(t, rpcUrl, statusParams)
	resMap := statusResp.Result.(map[string]interface{})

	// Check if Rclone metadata was injected (proof DB is working)
	rcloneMeta, ok := resMap["rclone"].(map[string]interface{})
	if !ok {
		t.Fatalf("Rclone metadata missing from tellStatus response. DB lookup failed?")
	}
	if rcloneMeta["targetRemote"] != "my-remote:" {
		t.Errorf("Expected remote 'my-remote:', got %v", rcloneMeta["targetRemote"])
	}
	t.Log("Verified Task Persistence in DB via API")

	// 5. Test: Add Torrent
	// Mock a small torrent or use magnet
	// Magnet is easier
	magnet := "magnet:?xt=urn:btih:88594aaacbde40ef3e2510c47374ec0aa396c08e&dn=test&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce"
	addMagnetParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.addUri", // Magnets use addUri
		"id":      "test-3",
		"params": []interface{}{
			"token:" + Aria2Secret,
			[]string{magnet},
			map[string]string{
				"rclone-target": "magnet-remote:",
			},
		},
	}
	respMagnet := sendRPC(t, rpcUrl, addMagnetParams)
	gidMagnet := respMagnet.Result.(string)
	t.Logf("Added Magnet GID: %s", gidMagnet)

	// 6. Simulate Restart
	t.Log("Restarting Server...")
	serverCmd.Process.Signal(os.Interrupt)
	serverCmd.Wait()

	// Give it a moment to cleanup
	time.Sleep(3 * time.Second)

	// Start again
	serverCmd2 := exec.Command("./server_test")
	// serverCmd2.Stdout = os.Stdout // Keep output clean
	serverCmd2.Env = serverEnv // Inherit mock HOME
	if err := serverCmd2.Start(); err != nil {
		t.Fatalf("Failed to restart server: %v", err)
	}
	defer func() {
		if serverCmd2.Process != nil {
			serverCmd2.Process.Signal(os.Interrupt)
			serverCmd2.Wait()
		}
	}()

	time.Sleep(5 * time.Second) // Wait for restore loop

	// 7. Verify Restoration
	// We check if the tasks exist in Aria2.
	// If restoration worked, tellStatus should return them.

	// Check HTTP Task
	statusParams["params"] = []interface{}{"token:" + Aria2Secret, gid}
	statusResp2 := sendRPC(t, rpcUrl, statusParams)
	if statusResp2.Error != nil {
		// If 404, restoration failed
		t.Fatalf("Task %s lost after restart! Restoration failed.", gid)
	}
	t.Logf("Task %s survived restart.", gid)

	// 8. Test: Change Option Persistence
	t.Log("Testing Change Option Persistence...")
	// Add a paused task so we can change options easily without it finishing
	pausedTaskParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.addUri",
		"id":      "test-opt-1",
		"params": []interface{}{
			"token:" + Aria2Secret,
			[]string{"https://mmatechnical.com/Download/Download-Test-File/(MMA)-100MB.zip"},
			map[string]string{
				"pause": "true",
				"split": "2",
			},
		},
	}
	respPaused := sendRPC(t, rpcUrl, pausedTaskParams)
	gidPaused := respPaused.Result.(string)
	t.Logf("Added Paused Task: %s", gidPaused)

	// Change Option: split=5
	changeOptParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.changeOption",
		"id":      "test-opt-2",
		"params": []interface{}{
			"token:" + Aria2Secret,
			gidPaused,
			map[string]string{
				"split": "5",
			},
		},
	}
	respChange := sendRPC(t, rpcUrl, changeOptParams)
	if respChange.Error != nil {
		t.Fatalf("Failed to change option: %v", respChange.Error)
	}
	t.Log("Changed option 'split' to 5")

	// 9. Test: Remove Task Persistence
	t.Log("Testing Remove Persistence...")
	// Add temporary task
	tempParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.addUri",
		"id":      "test-rm-1",
		"params": []interface{}{
			"token:" + Aria2Secret,
			[]string{"https://mmatechnical.com/Download/Download-Test-File/(MMA)-100MB.zip"},
		},
	}
	respTemp := sendRPC(t, rpcUrl, tempParams)
	gidTemp := respTemp.Result.(string)

	// Remove it
	removeParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.remove",
		"id":      "test-rm-2",
		"params": []interface{}{
			"token:" + Aria2Secret,
			gidTemp,
		},
	}
	rmRpcResp := sendRPC(t, rpcUrl, removeParams)
	if rmRpcResp.Error != nil {
		t.Fatalf("Failed to remove task %s: %v", gidTemp, rmRpcResp.Error)
	}
	t.Logf("Removed Task: %s", gidTemp)

	// 10. Restart Again
	t.Log("Restarting Server (Round 2)...")
	serverCmd2.Process.Signal(os.Interrupt)
	serverCmd2.Wait()
	time.Sleep(3 * time.Second)

	serverCmd3 := exec.Command("./server_test")
	serverCmd3.Env = serverEnv
	serverCmd3.Stdout = os.Stdout // Enable logs
	serverCmd3.Stderr = os.Stderr
	if err := serverCmd3.Start(); err != nil {
		t.Fatalf("Failed to restart server 3: %v", err)
	}
	defer func() {
		if serverCmd3.Process != nil {
			serverCmd3.Process.Signal(os.Interrupt)
			serverCmd3.Wait()
		}
	}()
	time.Sleep(5 * time.Second)

	// 11. Verify Options Restored
	optStatusParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.getOption",
		"id":      "test-check-opt",
		"params": []interface{}{
			"token:" + Aria2Secret,
			gidPaused,
		},
	}
	optResp := sendRPC(t, rpcUrl, optStatusParams)
	if optResp.Error != nil {
		t.Fatalf("Failed to get options for %s: %v", gidPaused, optResp.Error)
	}
	opts := optResp.Result.(map[string]interface{})
	if opts["split"] != "5" {
		t.Errorf("Expected split=5, got %v. Option persistence failed!", opts["split"])
	} else {
		t.Log("Option 'split=5' successfully restored.")
	}

	// 12. Verify Removal Persisted (Task should NOT exist)
	checkRmParams := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "aria2.tellStatus",
		"id":      "test-check-rm",
		"params": []interface{}{
			"token:" + Aria2Secret,
			gidTemp,
		},
	}
	rmResp := sendRPC(t, rpcUrl, checkRmParams)
	if rmResp.Error == nil {
		t.Errorf("Task %s should have been removed, but tellStatus found it!", gidTemp)
	} else {
		t.Logf("Task %s correctly missing after restart.", gidTemp)
	}
}

type JsonRpcResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Id      interface{} `json:"id"`
}

func sendRPC(t *testing.T, url string, reqData map[string]interface{}) JsonRpcResponse {
	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Prevent compression issues during test
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP Post failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp JsonRpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	return rpcResp
}
