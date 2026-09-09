// Test FTP filesystem interface
package ftp_test

import (
	"testing"

	"zclone/backend/ftp"
	"zclone/fstest"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against zclone FTP server
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFTPZclone:",
		NilObject:  (*ftp.Object)(nil),
	})
}

// TestIntegrationProftpd runs integration tests against proFTPd
func TestIntegrationProftpd(t *testing.T) {
	if *fstest.RemoteName != "" {
		t.Skip("skipping as -remote is set")
	}
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFTPProftpd:",
		NilObject:  (*ftp.Object)(nil),
	})
}

// TestIntegrationPureftpd runs integration tests against pureFTPd
func TestIntegrationPureftpd(t *testing.T) {
	if *fstest.RemoteName != "" {
		t.Skip("skipping as -remote is set")
	}
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFTPPureftpd:",
		NilObject:  (*ftp.Object)(nil),
	})
}

// TestIntegrationVsftpd runs integration tests against vsFTPd
func TestIntegrationVsftpd(t *testing.T) {
	if *fstest.RemoteName != "" {
		t.Skip("skipping as -remote is set")
	}
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFTPVsftpd:",
		NilObject:  (*ftp.Object)(nil),
	})
}
