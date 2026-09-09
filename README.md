# Zclone

Zclone is a local-first, independently buildable cloud-storage synchronisation tool. It is a command-line program to sync files and
directories to and from different cloud storage providers.

Build it offline with `make zclone`; all Go dependencies are included in `vendor/`.
This distribution does not assume a central website, download service, forum, or self-update service.
Run `make verify-local` to compile, type-check every package, and produce a local
CycloneDX SBOM plus SHA-256 source manifest in `build/`.

## Storage providers

- 1Fichier [:page_facing_up:](//fichier/)
- Akamai Netstorage [:page_facing_up:](//netstorage/)
- Alibaba Cloud (Aliyun) Object Storage System (OSS) [:page_facing_up:](//s3/#alibaba-oss)
- Amazon S3 [:page_facing_up:](//s3/)
- ArvanCloud Object Storage (AOS) [:page_facing_up:](//s3/#arvan-cloud-object-storage-aos)
- Bizfly Cloud Simple Storage [:page_facing_up:](//s3/#bizflycloud)
- Backblaze B2 [:page_facing_up:](//b2/)
- Box [:page_facing_up:](//box/)
- Ceph [:page_facing_up:](//s3/#ceph)
- China Mobile Ecloud Elastic Object Storage (EOS) [:page_facing_up:](//s3/#china-mobile-ecloud-eos)
- Citrix ShareFile [:page_facing_up:](//sharefile/)
- Cloudflare R2 [:page_facing_up:](//s3/#cloudflare-r2)
- Cloudinary [:page_facing_up:](//cloudinary/)
- Cubbit DS3 [:page_facing_up:](//s3/#Cubbit)
- DigitalOcean Spaces [:page_facing_up:](//s3/#digitalocean-spaces)
- Digi Storage [:page_facing_up:](//koofr/#digi-storage)
- Dreamhost [:page_facing_up:](//s3/#dreamhost)
- Drime [:page_facing_up:](//s3/#drime)
- Dropbox [:page_facing_up:](//dropbox/)
- Enterprise File Fabric [:page_facing_up:](//filefabric/)
- Exaba [:page_facing_up:](//s3/#exaba)
- Fastly Object Storage [:page_facing_up:](//s3/#fastly)
- Fastmail Files [:page_facing_up:](//webdav/#fastmail-files)
- FileLu [:page_facing_up:](//filelu/)
- Filen [:page_facing_up:](//filen/)
- Files.com [:page_facing_up:](//filescom/)
- FlashBlade [:page_facing_up:](//s3/#pure-storage-flashblade)
- FTP [:page_facing_up:](//ftp/)
- GoFile [:page_facing_up:](//gofile/)
- Google Cloud Storage [:page_facing_up:](//googlecloudstorage/)
- Google Drive [:page_facing_up:](//drive/)
- Google Photos [:page_facing_up:](//googlephotos/)
- HDFS (Hadoop Distributed Filesystem) [:page_facing_up:](//hdfs/)
- Hetzner Object Storage [:page_facing_up:](//s3/#hetzner)
- Hetzner Storage Box [:page_facing_up:](//sftp/#hetzner-storage-box)
- HiDrive [:page_facing_up:](//hidrive/)
- Hitachi Content Platform (HCP) [:page_facing_up:](//s3/#hcp)
- HTTP [:page_facing_up:](//http/)
- Huawei Cloud Object Storage Service(OBS) [:page_facing_up:](//s3/#huawei-obs)
- Huawei Drive [:page_facing_up:](//huaweidrive/)
- iCloud Drive [:page_facing_up:](//iclouddrive/)
- ImageKit [:page_facing_up:](//imagekit/)
- Internet Archive [:page_facing_up:](//internetarchive/)
- Internxt [:page_facing_up:](//internxt/)
- Jottacloud [:page_facing_up:](//jottacloud/)
- IBM COS S3 [:page_facing_up:](//s3/#ibm-cos-s3)
- Impossible Cloud [:page_facing_up:](//s3/#impossible-cloud)
- Intercolo Object Storage [:page_facing_up:](//s3/#intercolo)
- IONOS Cloud [:page_facing_up:](//s3/#ionos)
- Koofr [:page_facing_up:](//koofr/)
- Leviia Object Storage [:page_facing_up:](//s3/#leviia)
- Liara Object Storage [:page_facing_up:](//s3/#liara-object-storage)
- Linkbox [:page_facing_up:](//linkbox)
- Linode Object Storage [:page_facing_up:](//s3/#linode)
- Magalu Object Storage [:page_facing_up:](//s3/#magalu)
- Mail.ru Cloud [:page_facing_up:](//mailru/)
- Memset Memstore [:page_facing_up:](//swift/)
- MEGA [:page_facing_up:](//mega/)
- MEGA S4 Object Storage [:page_facing_up:](//s3/#mega)
- Memory [:page_facing_up:](//memory/)
- Microsoft Azure Blob Storage [:page_facing_up:](//azureblob/)
- Microsoft Azure Files Storage [:page_facing_up:](//azurefiles/)
- Microsoft OneDrive [:page_facing_up:](//onedrive/)
- Minio [:page_facing_up:](//s3/#minio)
- Nextcloud [:page_facing_up:](//webdav/#nextcloud)
- Blomp Cloud Storage [:page_facing_up:](//swift/)
- OpenDrive [:page_facing_up:](//opendrive/)
- OpenStack Swift [:page_facing_up:](//swift/)
- Oracle Cloud Storage [:page_facing_up:](//swift/)
- Oracle Object Storage [:page_facing_up:](//oracleobjectstorage/)
- Outscale [:page_facing_up:](//s3/#outscale)
- OVHcloud Object Storage (Swift) [:page_facing_up:](//swift/)
- OVHcloud Object Storage (S3-compatible) [:page_facing_up:](//s3/#ovhcloud)
- ownCloud [:page_facing_up:](//webdav/#owncloud)
- pCloud [:page_facing_up:](//pcloud/)
- Petabox [:page_facing_up:](//s3/#petabox)
- PikPak [:page_facing_up:](//pikpak/)
- Pixeldrain [:page_facing_up:](//pixeldrain/)
- premiumize.me [:page_facing_up:](//premiumizeme/)
- put.io [:page_facing_up:](//putio/)
- Proton Drive [:page_facing_up:](//protondrive/)
- QingStor [:page_facing_up:](//qingstor/)
- Qiniu Cloud Object Storage (Kodo) [:page_facing_up:](//s3/#qiniu)
- Rabata Cloud Storage [:page_facing_up:](//s3/#Rabata)
- Quatrix [:page_facing_up:](//quatrix/)
- Rackspace Cloud Files [:page_facing_up:](//swift/)
- RackCorp Object Storage [:page_facing_up:](//s3/#RackCorp)
- rsync.net [:page_facing_up:](//sftp/#rsync-net)
- Scaleway [:page_facing_up:](//s3/#scaleway)
- Scality (RING / ARTESCA) [:page_facing_up:](//s3/#scality)
- Seafile [:page_facing_up:](//seafile/)
- Seagate Lyve Cloud [:page_facing_up:](//s3/#lyve)
- SeaweedFS [:page_facing_up:](//s3/#seaweedfs)
- Selectel Object Storage [:page_facing_up:](//s3/#selectel)
- Servercore Object Storage [:page_facing_up:](//s3/#servercore)
- SFTP [:page_facing_up:](//sftp/)
- Shade [:page_facing_up:](//shade/)
- SMB / CIFS [:page_facing_up:](//smb/)
- Spectra Logic [:page_facing_up:](//s3/#spectralogic)
- Storj [:page_facing_up:](//storj/)
- SugarSync [:page_facing_up:](//sugarsync/)
- Synology C2 Object Storage [:page_facing_up:](//s3/#synology-c2)
- Tencent Cloud Object Storage (COS) [:page_facing_up:](//s3/#tencent-cos)
- Uloz.to [:page_facing_up:](//ulozto/)
- US3 Object Storage [:page_facing_up:](//s3/#us3)
- Wasabi [:page_facing_up:](//s3/#wasabi)
- WebDAV [:page_facing_up:](//webdav/)
- Yandex Disk [:page_facing_up:](//yandex/)
- Zadara Object Storage [:page_facing_up:](//s3/#zadara)
- Zero Services (ZERO-Z3) [:page_facing_up:](//s3/#zero-z3)
- Zoho WorkDrive [:page_facing_up:](//zoho/)
- Zata.ai [:page_facing_up:](//s3/#Zata)
- The local filesystem [:page_facing_up:](//local/)

Please see [the full list of all storage providers and their features](//overview/)

### Virtual storage providers

These backends adapt or modify other storage providers

- Alias: rename existing remotes [:page_facing_up:](//alias/)
- Archive: read archive files [:page_facing_up:](//archive/)
- Cache: cache remotes (DEPRECATED) [:page_facing_up:](//cache/)
- Chunker: split large files [:page_facing_up:](//chunker/)
- Combine: combine multiple remotes into a directory tree [:page_facing_up:](//combine/)
- Compress: compress files [:page_facing_up:](//compress/)
- Crypt: encrypt files [:page_facing_up:](//crypt/)
- Hasher: hash files [:page_facing_up:](//hasher/)
- Union: join multiple remotes to work together [:page_facing_up:](//union/)

## Features

- MD5/SHA-1 hashes checked at all times for file integrity
- Timestamps preserved on files
- Partial syncs supported on a whole file basis
- [Copy](//commands/zclone_copy/) mode to just copy new/changed
  files
- [Sync](//commands/zclone_sync/) (one way) mode to make a directory
  identical
- [Bisync](//bisync/) (two way) to keep two directories in sync
  bidirectionally
- [Check](//commands/zclone_check/) mode to check for file hash
  equality
- Can sync to and from network, e.g. two different cloud accounts
- Optional large file chunking ([Chunker](//chunker/))
- Optional transparent compression ([Compress](//compress/))
- Optional encryption ([Crypt](//crypt/))
- Optional FUSE mount ([zclone mount](//commands/zclone_mount/))
- Multi-threaded downloads to local disk
- Can [serve](//commands/zclone_serve/) local or remote files
  over HTTP/WebDAV/FTP/SFTP/DLNA

## Installation & documentation

Please see the [zclone website](//) for:

- [Installation](//install/)
- [Documentation & configuration](//docs/)
- [Changelog](//changelog/)
- [FAQ](//faq/)
- [Storage providers](//overview/)
- [Forum](/)
- ...and more

## Downloads

- <//downloads/>

## License

This is free software under the terms of the MIT license (check the
[COPYING file](/COPYING) included in this package).
