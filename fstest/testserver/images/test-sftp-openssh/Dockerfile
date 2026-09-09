# A very minimal sftp server for integration testing zclone
FROM alpine:latest

# User zclone, password password
RUN \
    apk add openssh && \
    ssh-keygen -A && \
    adduser -D zclone && \
    echo "zclone:password" | chpasswd

ENTRYPOINT [ "/usr/sbin/sshd", "-D" ]
