package service

// StopGracefully signals background routines to stop
func (s *DownloadService) StopGracefully() {
	close(s.stop)
	// We could also wait for routines to finish if we tracked them with WaitGroup
}
