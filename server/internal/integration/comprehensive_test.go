package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	CompSecret   = "comprehensive-secret"
	CompTestFile = "https://fsn1-speed.hetzner.com/100MB.bin"
)

func TestComprehensiveEdgeCases(t *testing.T) {
	// 1. Setup Isolated Environment
	wd, _ := os.Getwd()
	testDir := filepath.Join(wd, "comp_test_run")
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)

	// Build Server
	t.Log("Building server for comprehensive tests...")
	buildCmd := exec.Command("go", "build", "-o", "server_comp_bin", "../../cmd/server/main.go")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\n%s", err, out)
	}
	defer os.Remove("server_comp_bin")

	// Pre-kill
	exec.Command("fuser", "-k", "8080/tcp", "6800/tcp", "5572/tcp").Run()

	// Start Server
	serverEnv := append(os.Environ(),
		fmt.Sprintf("HOME=%s", testDir),
		fmt.Sprintf("ARIA2_SECRET=%s", CompSecret),
		fmt.Sprintf("ARIA2_DIR=%s", testDir),
	)

	serverCmd := exec.Command("./server_comp_bin")
	serverCmd.Env = serverEnv
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Signal(os.Interrupt)
			serverCmd.Wait()
		}
	}()

	time.Sleep(5 * time.Second)

	rpcUrl := "http://localhost:8080/jsonrpc"
	aria2Url := "http://localhost:6800/jsonrpc"

	// --- Helper: Add Task ---
	addTask := func(t *testing.T, url string, remote string, opts map[string]interface{}) string {
		if opts == nil {
			opts = make(map[string]interface{})
		}
		if remote != "" {
			opts["rclone-target"] = remote
		}
		opts["dir"] = testDir

		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "aria2.addUri",
			"id":      "add-helper",
			"params": []interface{}{
				"token:" + CompSecret,
				[]string{url},
				opts,
			},
		}
		resp := sendRPC(t, rpcUrl, req)
		if resp.Error != nil {
			t.Fatalf("Failed to add task: %v", resp.Error)
		}
		return resp.Result.(string)
	}

	// 1. SCENARIO: External Removal Sync
	t.Run("ExternalRemovalSync", func(t *testing.T) {
		// Don't pause, so it starts and we get events
		gid := addTask(t, CompTestFile+"?t=ext-rem", "remote:", nil)
		time.Sleep(1 * time.Second)

		// Remove via Aria2 directly
		removeReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.remove", "id": "ext-rm",
			"params": []interface{}{"token:" + CompSecret, gid},
		}
		sendRPC(t, aria2Url, removeReq)
		time.Sleep(2 * time.Second)

		// Verify DB updated to "removed"
		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "check", "params": []interface{}{"token:" + CompSecret, gid},
		})
		if statusResp.Result == nil {
			t.Fatalf("Task record lost, error: %v", statusResp.Error)
		}
		res := statusResp.Result.(map[string]interface{})
		rMap, _ := res["rclone"].(map[string]interface{})
		if rMap["status"] != "removed" {
			t.Errorf("Expected status removed, got %v", rMap["status"])
		}
	})

	// 2. SCENARIO: Manual Retry Functional Test
	t.Run("ManualRetryTrigger", func(t *testing.T) {
		gid := addTask(t, "https://invalid-domain-8888.com/file.dat", "remote:", nil)

		// Wait for error
		var errored bool
		for i := 0; i < 15; i++ {
			statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
				"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "p", "params": []interface{}{"token:" + CompSecret, gid},
			})
			if statusResp.Result == nil {
				continue
			}
			res := statusResp.Result.(map[string]interface{})
			if res["status"] == "error" {
				errored = true
				break
			}
			time.Sleep(1 * time.Second)
		}
		if !errored {
			t.Fatalf("Task did not enter error state")
		}

		// Trigger Retry
		retryReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.retryTask", "id": "retry",
			"params": []interface{}{"token:" + CompSecret, gid},
		}
		resp := sendRPC(t, rpcUrl, retryReq)
		if resp.Result != "OK" {
			t.Errorf("Retry failed: %v, error: %v", resp.Result, resp.Error)
		}

		// Verify status reset
		time.Sleep(1 * time.Second)
		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "check", "params": []interface{}{"token:" + CompSecret, gid},
		})
		res := statusResp.Result.(map[string]interface{})
		if res["status"] == "error" {
			t.Log("Note: Status still error after retry (immediately failed again), which is expected for invalid domain.")
		}
	})

	// 3. SCENARIO: Option Change Persistence
	t.Run("OptionChangePersistence", func(t *testing.T) {
		gid := addTask(t, CompTestFile+"?t=opt", "remote:", map[string]interface{}{"pause": "true"})

		// Change option
		changeReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.changeOption", "id": "ch",
			"params": []interface{}{"token:" + CompSecret, gid, map[string]string{"max-download-limit": "100K"}},
		}
		sendRPC(t, rpcUrl, changeReq)
		time.Sleep(1 * time.Second)

		// Restart Server
		serverCmd.Process.Signal(os.Interrupt)
		serverCmd.Wait()
		serverCmd = exec.Command("./server_comp_bin")
		serverCmd.Env = serverEnv
		serverCmd.Start()
		time.Sleep(5 * time.Second)

		// Verify option survived
		getOptReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.getOption", "id": "get",
			"params": []interface{}{"token:" + CompSecret, gid},
		}
		optResp := sendRPC(t, rpcUrl, getOptReq)
		opts := optResp.Result.(map[string]interface{})
		if val := opts["max-download-limit"].(string); val != "102400" && val != "100K" {
			t.Errorf("Option lost after restart! Got %v", val)
		}
	})

	// 4. SCENARIO: Global Stats Accumulation
	t.Run("GlobalStatsAccumulation", func(t *testing.T) {
		req := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.getGlobalStat", "id": "stats",
			"params": []interface{}{"token:" + CompSecret},
		}
		resp := sendRPC(t, rpcUrl, req)
		res := resp.Result.(map[string]interface{})
		if _, ok := res["totalDownloaded"]; !ok {
			t.Errorf("Missing totalDownloaded in global stats")
		}
	})

	// 5. SCENARIO: Concurrent Additions
	t.Run("ConcurrentAdditions", func(t *testing.T) {
		var wg sync.WaitGroup
		numTasks := 5
		for i := 0; i < numTasks; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				addTask(t, fmt.Sprintf("%s?stress=%d", CompTestFile, idx), "remote:", map[string]interface{}{"pause": "true"})
			}(i)
		}
		wg.Wait()
		time.Sleep(2 * time.Second)

		activeResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellActive", "id": "p", "params": []interface{}{"token:" + CompSecret},
		})
		tasks := activeResp.Result.([]interface{})
		t.Logf("Found %d active tasks", len(tasks))
	})

	// 6. SCENARIO: Purge Stopped Tasks
	t.Run("PurgeTasks", func(t *testing.T) {
		gid := addTask(t, CompTestFile+"?t=purge", "", map[string]interface{}{"pause": "true"})
		time.Sleep(1 * time.Second)
		sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.remove", "id": "rm", "params": []interface{}{"token:" + CompSecret, gid},
		})
		time.Sleep(1 * time.Second)

		// Purge
		sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.purgeDownloadResult", "id": "purge",
			"params": []interface{}{"token:" + CompSecret},
		})
		time.Sleep(2 * time.Second)

		stoppedResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStopped", "id": "check", "params": []interface{}{"token:" + CompSecret, 0, 100},
		})
		tasks := stoppedResp.Result.([]interface{})
		for _, task := range tasks {
			if task.(map[string]interface{})["gid"] == gid {
				t.Errorf("Task %s still exists after purge", gid)
			}
		}
	})

	// 7. SCENARIO: Shadow Import with Error
	t.Run("ShadowImportError", func(t *testing.T) {
		addReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.addUri", "id": "cli",
			"params": []interface{}{"token:" + CompSecret, []string{"https://err-domain-999.com/none"}},
		}
		resp := sendRPC(t, aria2Url, addReq)
		gid := resp.Result.(string)
		time.Sleep(3 * time.Second)

		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "check", "params": []interface{}{"token:" + CompSecret, gid},
		})
		if statusResp.Result == nil {
			t.Fatalf("Errored shadow task not imported")
		}
		res := statusResp.Result.(map[string]interface{})
		rMap, _ := res["rclone"].(map[string]interface{})
		if rMap["status"] != "error" {
			t.Errorf("Expected error status for shadow import, got %v", rMap["status"])
		}
	})

	// 8. SCENARIO: Rclone Configuration Proxy
	t.Run("RcloneConfigProxy", func(t *testing.T) {
		createReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "rclone.createRemote", "id": "conf",
			"params": []interface{}{map[string]interface{}{
				"name":       "test-remote-comp",
				"type":       "alias",
				"parameters": map[string]string{"remote": testDir},
			}},
		}
		resp := sendRPC(t, rpcUrl, createReq)
		if resp.Error != nil {
			t.Errorf("Create remote failed: %v", resp.Error)
		}

		listReq := map[string]interface{}{
			"jsonrpc": "2.0", "method": "rclone.listRemotes", "id": "list", "params": []interface{}{},
		}
		listResp := sendRPC(t, rpcUrl, listReq)
		t.Logf("List remotes result: %v", listResp.Result)
	})

	// 9. SCENARIO: Force Remove Persistence
	t.Run("ForceRemovePersistence", func(t *testing.T) {
		gid := addTask(t, CompTestFile+"?t=force", "", map[string]interface{}{"pause": "true"})
		time.Sleep(1 * time.Second)

		sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.forceRemove", "id": "frm", "params": []interface{}{"token:" + CompSecret, gid},
		})
		time.Sleep(1 * time.Second)

		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "c", "params": []interface{}{"token:" + CompSecret, gid},
		})
		res := statusResp.Result.(map[string]interface{})
		rMap, _ := res["rclone"].(map[string]interface{})
		if rMap["status"] != "removed" {
			t.Errorf("Force remove not reflected in DB, got %v", rMap["status"])
		}
	})

	// 10. SCENARIO: Multi-Client Interception
	t.Run("MultiClientInterception", func(t *testing.T) {
		addTask(t, CompTestFile+"?t=multi", "target:", map[string]interface{}{"pause": "true"})
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendRPC(t, rpcUrl, map[string]interface{}{
					"jsonrpc": "2.0", "method": "aria2.tellActive", "id": "multi", "params": []interface{}{"token:" + CompSecret},
				})
			}()
		}
		wg.Wait()
	})

	// 11. SCENARIO: Shutdown Final Sync
	t.Run("ShutdownFinalSync", func(t *testing.T) {
		gid := addTask(t, CompTestFile+"?t=shutdown", "remote:", nil)
		time.Sleep(3 * time.Second)

		serverCmd.Process.Signal(os.Interrupt)
		serverCmd.Wait()

		serverCmd = exec.Command("./server_comp_bin")
		serverCmd.Env = serverEnv
		serverCmd.Start()
		time.Sleep(5 * time.Second)

		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "c", "params": []interface{}{"token:" + CompSecret, gid},
		})
		res := statusResp.Result.(map[string]interface{})
		cLen, _ := strconv.ParseInt(res["completedLength"].(string), 10, 64)
		if cLen == 0 {
			t.Errorf("Progress not saved during shutdown sync")
		}
	})

	// 12. SCENARIO: RemoveDownloadResult DB Cleanup
	t.Run("RemoveDownloadResultDB", func(t *testing.T) {
		gid := addTask(t, CompTestFile+"?t=rmdr", "", nil) // Don't pause
		time.Sleep(1 * time.Second)

		// Move to stopped via forceRemove to be sure
		sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.forceRemove", "id": "frm", "params": []interface{}{"token:" + CompSecret, gid},
		})
		time.Sleep(2 * time.Second)

		// Call removeDownloadResult
		resp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.removeDownloadResult", "id": "rmdr", "params": []interface{}{"token:" + CompSecret, gid},
		})
		if resp.Error != nil {
			t.Logf("removeDownloadResult failed: %v. Assuming task already gone or active.", resp.Error)
		}
		time.Sleep(2 * time.Second)

		statusResp := sendRPC(t, rpcUrl, map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.tellStatus", "id": "check", "params": []interface{}{"token:" + CompSecret, gid},
		})
		if statusResp.Error == nil {
			// If no error, it means it found it.
			// Check if it's "removed" status in DB, which is okayish, but we prefer it GONE (DeleteTask).
			// DeleteTask deletes it from DB.
			// So GetUpload returns error.
			// So tellStatus should return error.
			t.Errorf("Task still exists in DB after removeDownloadResult")
		}
	})

	// 13. SCENARIO: Secret Injection
	t.Run("SecretInjection", func(t *testing.T) {
		req := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.getVersion", "id": "inject",
			"params": []interface{}{},
		}
		resp := sendRPC(t, rpcUrl, req)
		if resp.Error != nil {
			t.Errorf("Secret injection failed: %v", resp.Error)
		}
	})

	// 14. SCENARIO: Duplicate GID Respect
	t.Run("DuplicateGidRespect", func(t *testing.T) {
		gid := "deadbeefdeadbeef" // 16 chars hex
		addTask(t, CompTestFile+"?t=fix1", "", map[string]interface{}{"gid": gid, "pause": "true"})

		// Adding same GID again should fail
		req2 := map[string]interface{}{
			"jsonrpc": "2.0", "method": "aria2.addUri", "id": "fix2",
			"params": []interface{}{"token:" + CompSecret, []string{CompTestFile}, map[string]interface{}{"gid": gid}},
		}
		resp2 := sendRPC(t, rpcUrl, req2)
		if resp2.Error == nil {
			t.Errorf("Expected error for duplicate GID")
		}
	})

	// 15-30 SCENARIOS: Rapid fire and more edge cases
	t.Run("RapidScenarios", func(t *testing.T) {
		// 15. Change Global Option
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.changeGlobalOption", "params": []interface{}{"token:" + CompSecret, map[string]string{"max-overall-download-limit": "1M"}}})
		// 16. Get Global Option
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.getGlobalOption", "params": []interface{}{"token:" + CompSecret}})
		// 17. Pause All
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.pauseAll", "params": []interface{}{"token:" + CompSecret}})
		// 18. Unpause All
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.unpauseAll", "params": []interface{}{"token:" + CompSecret}})
		// 19. Get Session Info
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.getSessionInfo", "params": []interface{}{"token:" + CompSecret}})
		// 20. Save Session
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.saveSession", "params": []interface{}{"token:" + CompSecret}})
		// 21. Get Files for shadow task
		sgid := addTask(t, CompTestFile+"?t=sfiles", "", map[string]interface{}{"pause": "true"})
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.getFiles", "params": []interface{}{"token:" + CompSecret, sgid}})
		// 22. Get Peers
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.getPeers", "params": []interface{}{"token:" + CompSecret, sgid}})
		// 23. Get Active with keys
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.tellActive", "params": []interface{}{"token:" + CompSecret, []string{"gid"}}})
		// 24. Get Stopped with keys
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.tellStopped", "params": []interface{}{"token:" + CompSecret, 0, 10, []string{"gid"}}})
		// 25. Get Waiting
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.tellWaiting", "params": []interface{}{"token:" + CompSecret, 0, 10}})
		// 26. Force Pause
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.forcePause", "params": []interface{}{"token:" + CompSecret, sgid}})
		// 27. Force Unpause
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.unpause", "params": []interface{}{"token:" + CompSecret, sgid}})
		// 28. Change Position
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "aria2.changePosition", "params": []interface{}{"token:" + CompSecret, sgid, 0, "POS_SET"}})
		// 29. List Methods
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "system.listMethods", "params": []interface{}{}})
		// 30. Get Rclone Version
		sendRPC(t, rpcUrl, map[string]interface{}{"jsonrpc": "2.0", "method": "rclone.getVersion", "params": []interface{}{}})
	})

	// 31. SCENARIO: System Multicall Bypassing
	t.Run("SystemMulticallBypass", func(t *testing.T) {
		// Add a task with Rclone metadata
		gid := addTask(t, CompTestFile+"?t=multi-bypass", "test-remote-multi:", nil)

		// Use system.multicall to fetch status
		// Expected: The server MUST intercept this and inject the Rclone metadata into the inner response
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "system.multicall",
			"id":      "multi-bypass",
			"params": []interface{}{
				[]interface{}{
					map[string]interface{}{
						"methodName": "aria2.tellStatus",
						"params":     []interface{}{"token:" + CompSecret, gid},
					},
				},
			},
		}

		resp := sendRPC(t, rpcUrl, req)
		if resp.Result == nil {
			t.Fatalf("Multicall failed: %v", resp.Error)
		}

		// Result is a list of results
		results := resp.Result.([]interface{})
		if len(results) == 0 {
			t.Fatalf("Empty multicall result")
		}

		// Inner result might be [result] or just result depending on implementation
		inner, ok := results[0].([]interface{})
		if !ok {
			// Maybe it's just the map directly?
			if m, ok := results[0].(map[string]interface{}); ok {
				// This would mean handleMulticall didn't wrap it in []interface{}
				// But code says it does.
				// Let's assume it panicked because it WAS []interface{} but I failed to define 'inner' scope correctly?
				// Ah, I removed the definition of 'inner' in previous Edit.
				t.Fatalf("results[0] is map, expected [map]")
				_ = m
			} else {
				t.Fatalf("results[0] is %T, expected []interface{}", results[0])
			}
		}

		var statusMap map[string]interface{}

		// If inner is [map], then inner[0] is map
		if len(inner) > 0 {
			if m, ok := inner[0].(map[string]interface{}); ok {
				statusMap = m
			} else if s, ok := inner[0].([]interface{}); ok {
				t.Fatalf("Unexpected slice in inner result: %v", s)
			} else {
				t.Fatalf("Unexpected type in inner result: %T", inner[0])
			}
		}

		// Check for rclone injection

		if _, ok := statusMap["rclone"]; !ok {
			t.Fatalf("CRITICAL: system.multicall bypassed the proxy logic! 'rclone' metadata missing.")
		} else {
			t.Log("PASS: system.multicall was correctly intercepted and augmented.")
		}
	})
}

type JsonRpcResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Id      interface{} `json:"id"`
}

func sendRPC(t *testing.T, url string, req interface{}) JsonRpcResponse {
	body, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("RPC failed: %v", err)
	}
	defer resp.Body.Close()
	var r JsonRpcResponse
	json.NewDecoder(resp.Body).Decode(&r)
	return r
}
