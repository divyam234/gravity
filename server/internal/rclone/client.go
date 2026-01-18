package rclone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Client struct {
	url string
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

func (c *Client) Call(method string, params interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.url, method), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rclone error: %v", result)
	}

	return result, nil
}

func (c *Client) CopyFile(srcFs, srcRemote, dstFs, dstRemote string) error {
	params := map[string]string{
		"srcFs":     srcFs,
		"srcRemote": srcRemote,
		"dstFs":     dstFs,
		"dstRemote": dstRemote,
	}
	_, err := c.Call("operations/copyfile", params)
	return err
}

func (c *Client) CopyFileAsync(srcFs, srcRemote, dstFs, dstRemote string, customJobId int64) (string, error) {
	params := map[string]interface{}{
		"srcFs":     srcFs,
		"srcRemote": srcRemote,
		"dstFs":     dstFs,
		"dstRemote": dstRemote,
		"_async":    true,
		"_jobid":    customJobId,
	}
	res, err := c.Call("operations/copyfile", params)
	if err != nil {
		return "", err
	}
	// The response should contain 'jobid' which should match customJobId (as string or int)
	if jobid, ok := res["jobid"].(json.Number); ok {
		return jobid.String(), nil
	} else if jobidStr, ok := res["jobid"].(string); ok {
		return jobidStr, nil
	} else if jobidInt, ok := res["jobid"].(float64); ok { // json decode numbers as float64 by default
		return fmt.Sprintf("%.0f", jobidInt), nil
	}

	// If customJobId was respected, we can just return it stringified if response is missing it
	// But let's trust the response or fallback
	return fmt.Sprintf("%d", customJobId), nil
}

func (c *Client) JobStatus(jobId string) (string, error) {
	// job/status expects jobid as integer
	jobIdInt, err := strconv.ParseInt(jobId, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid job id: %w", err)
	}
	params := map[string]int64{"jobid": jobIdInt}
	res, err := c.Call("job/status", params)
	if err != nil {
		return "", err
	}

	finished, _ := res["finished"].(bool)
	success, _ := res["success"].(bool)
	errMsg, _ := res["error"].(string)

	if !finished {
		return "running", nil
	}
	if !success {
		return "failed", fmt.Errorf("%s", errMsg)
	}
	return "success", nil
}

func (c *Client) StopJob(jobId string) error {
	jobIdInt, err := strconv.ParseInt(jobId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid job id: %w", err)
	}
	params := map[string]int64{"jobid": jobIdInt}
	_, err = c.Call("job/stop", params)
	return err
}
