#!/usr/bin/env bash
exec zclone --check-normalization=true --check-control=true --check-length=true info \
	/tmp/testInfo \
	TestB2:testInfo \
	TestCryptDrive:testInfo \
	TestCryptSwift:testInfo \
	TestDrive:testInfo \
	TestDropbox:testInfo \
	TestGoogleCloudStorage:zclone-testinfo \
	TestnStorage:testInfo \
	TestOneDrive:testInfo \
	TestS3:zclone-testinfo \
	TestSftp:testInfo \
	TestSwift:testInfo \
	TestYandex:testInfo \
	TestFTP:testInfo
