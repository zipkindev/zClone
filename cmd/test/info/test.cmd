set ZCLONE_CONFIG_LOCALWINDOWS_TYPE=local
zclone.exe purge    LocalWindows:info
zclone.exe info -vv LocalWindows:info --write-json=info-LocalWindows.json > info-LocalWindows.log  2>&1
zclone.exe ls   -vv LocalWindows:info > info-LocalWindows.list 2>&1
