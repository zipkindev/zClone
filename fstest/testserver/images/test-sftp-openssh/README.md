# Test SFTP Openssh

This is a docker image for zclone's integration tests which runs an
openssh server in a docker image.

## Build

```
docker build --rm -t zclone/test-sftp-openssh .
docker push zclone/test-sftp-openssh
```

# Test

```
zclone lsf -R --sftp-host 172.17.0.2 --sftp-user zclone --sftp-pass $(zclone obscure password) :sftp:
```
