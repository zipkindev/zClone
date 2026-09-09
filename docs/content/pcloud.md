---
title: "pCloud"
description: "Zclone docs for pCloud"
versionIntroduced: "v1.39"
---

# pCloud

Paths are specified as `remote:path`

Paths may be as deep as required, e.g. `remote:directory/subdirectory`.

## Configuration

The initial setup for pCloud involves getting a token from pCloud which you
need to do in your browser.  `zclone config` walks you through it.

Here is an example of how to make a remote called `remote`.  First run:

```console
zclone config
```

This will guide you through an interactive setup process:

```text
No remotes found, make a new one?
n) New remote
s) Set configuration password
q) Quit config
n/s/q> n
name> remote
Type of storage to configure.
Choose a number from below, or type in your own value
[snip]
XX / Pcloud
   \ "pcloud"
[snip]
Storage> pcloud
Pcloud App Client Id - leave blank normally.
client_id> 
Pcloud App Client Secret - leave blank normally.
client_secret> 
Edit advanced config?
y) Yes
n) No (default)
y/n> n
Remote config
Use web browser to automatically authenticate zclone with remote?
 * Say Y if the machine running zclone has a web browser you can use
 * Say N if running zclone on a (remote) machine without web browser access
If not sure try Y. If Y failed, try N.
y) Yes
n) No
y/n> y
If your browser doesn't open automatically go to the following link: http://127.0.0.1:53682/auth
Log in and authorize zclone for access
Waiting for code...
Got code
Configuration complete.
Options:
- type: pcloud
- client_id:
- client_secret:
- token: {"access_token":"XXX","token_type":"bearer","expiry":"0001-01-01T00:00:00Z"}
Keep this "remote" remote?
y) Yes this is OK
e) Edit this remote
d) Delete this remote
y/e/d> y
```

See the [remote setup docs](/remote_setup/) for how to set it up on a
machine without an internet-connected web browser available.

Note if you are using remote config with zclone authorize while your pcloud
server is the EU region, you will need to set the hostname in 'Edit advanced
config', otherwise you might get a token error.

Note that zclone runs a webserver on your local machine to collect the
token as returned from pCloud. This only runs from the moment it opens
your browser to the moment you get back the verification code.  This
is on `http://127.0.0.1:53682/` and this it may require you to unblock
it temporarily if you are running a host firewall.

Once configured you can then use `zclone` like this (replace `remote` with the
name you gave your remote):

List directories in top level of your pCloud

```console
zclone lsd remote:
```

List all the files in your pCloud

```console
zclone ls remote:
```

To copy a local directory to a pCloud directory called backup

```console
zclone copy /home/source remote:backup
```

### Modification times and hashes

pCloud allows modification times to be set on objects accurate to 1
second.  These will be used to detect whether objects need syncing or
not.  In order to set a Modification time pCloud requires the object
be re-uploaded.

pCloud supports MD5 and SHA1 hashes in the US region, and SHA1 and SHA256
hashes in the EU region, so you can use the `--checksum` flag.

### Restricted filename characters

In addition to the [default restricted characters set](/overview/#restricted-characters)
the following characters are also replaced:

| Character | Value | Replacement |
| --------- |:-----:|:-----------:|
| \         | 0x5C  | ＼          |

Invalid UTF-8 bytes will also be [replaced](/overview/#invalid-utf8),
as they can't be used in JSON strings.

### Deleting files

Deleted files will be moved to the trash.  Your subscription level
will determine how long items stay in the trash.  `zclone cleanup` can
be used to empty the trash.

### Emptying the trash

Due to an API limitation, the `zclone cleanup` command will only work if you
set your username and password in the advanced options for this backend.
Since we generally want to avoid storing user passwords in the zclone config
file, we advise you to only set this up if you need the `zclone cleanup` command
to work.

### Root folder ID

You can set the `root_folder_id` for zclone.  This is the directory
(identified by its `Folder ID`) that zclone considers to be the root
of your pCloud drive.

Normally you will leave this blank and zclone will determine the
correct root to use itself.

However you can set this to restrict zclone to a specific folder
hierarchy.

In order to do this you will have to find the `Folder ID` of the
directory you wish zclone to display. This can be accomplished by executing
the ```zclone lsf``` command using a basic configuration setup that does not
include the ```root_folder_id``` parameter.

The command will enumerate available directories, allowing you to locate the
appropriate Folder ID for subsequent use.

Example:

```console
$ zclone lsf --dirs-only -Fip --csv TestPcloud:
dxxxxxxxx2,My Music/
dxxxxxxxx3,My Pictures/
dxxxxxxxx4,My Videos/
```

So if the folder you want zclone to use your is "My Music/", then use the returned
id from ```zclone lsf``` command (ex. `dxxxxxxxx2`) as the `root_folder_id` variable
value in the config file.

### Change notifications and mounts

The pCloud backend supports real‑time updates for zclone mounts via change
notifications. zclone uses pCloud’s diff long‑polling API to detect changes and
will automatically refresh directory listings in the mounted filesystem when
changes occur.

Notes and behavior:

- Works automatically when using `zclone mount` and requires no additional
  configuration.
- Notifications are directory‑scoped: when zclone detects a change, it refreshes
  the affected directory so new/removed/renamed files become visible promptly.
- Updates are near real‑time. The backend uses a long‑poll with short fallback
  polling intervals, so you should see changes appear quickly without manual
  refreshes.

If you want to debug or verify notifications, you can use the helper command:

```bash
zclone test changenotify remote:
```

This will log incoming change notifications for the given remote.

<!-- autogenerated options start - DO NOT EDIT - instead edit fs.RegInfo in backend/pcloud/pcloud.go and run make backenddocs to verify --> <!-- markdownlint-disable-line line-length -->
### Standard options

Here are the Standard options specific to pcloud (Pcloud).

#### --pcloud-client-id

OAuth Client Id.

Leave blank normally.

Properties:

- Config:      client_id
- Env Var:     ZCLONE_PCLOUD_CLIENT_ID
- Type:        string
- Required:    false

#### --pcloud-client-secret

OAuth Client Secret.

Leave blank normally.

Properties:

- Config:      client_secret
- Env Var:     ZCLONE_PCLOUD_CLIENT_SECRET
- Type:        string
- Required:    false

### Advanced options

Here are the Advanced options specific to pcloud (Pcloud).

#### --pcloud-token

OAuth Access Token as a JSON blob.

Properties:

- Config:      token
- Env Var:     ZCLONE_PCLOUD_TOKEN
- Type:        string
- Required:    false

#### --pcloud-auth-url

Auth server URL.

Leave blank to use the provider defaults.

Properties:

- Config:      auth_url
- Env Var:     ZCLONE_PCLOUD_AUTH_URL
- Type:        string
- Required:    false

#### --pcloud-token-url

Token server url.

Leave blank to use the provider defaults.

Properties:

- Config:      token_url
- Env Var:     ZCLONE_PCLOUD_TOKEN_URL
- Type:        string
- Required:    false

#### --pcloud-client-credentials

Use client credentials OAuth flow.

This will use the OAUTH2 client Credentials Flow as described in RFC 6749.

Note that this option is NOT supported by all backends.

Properties:

- Config:      client_credentials
- Env Var:     ZCLONE_PCLOUD_CLIENT_CREDENTIALS
- Type:        bool
- Default:     false

#### --pcloud-encoding

The encoding for the backend.

See the [encoding section in the overview](/overview/#encoding) for more info.

Properties:

- Config:      encoding
- Env Var:     ZCLONE_PCLOUD_ENCODING
- Type:        Encoding
- Default:     Slash,BackSlash,Del,Ctl,InvalidUtf8,Dot

#### --pcloud-root-folder-id

Fill in for zclone to use a non root folder as its starting point.

Properties:

- Config:      root_folder_id
- Env Var:     ZCLONE_PCLOUD_ROOT_FOLDER_ID
- Type:        string
- Default:     "d0"

#### --pcloud-hostname

Hostname to connect to.

This is normally set when zclone initially does the oauth connection,
however you will need to set it by hand if you are using remote config
with zclone authorize.


Properties:

- Config:      hostname
- Env Var:     ZCLONE_PCLOUD_HOSTNAME
- Type:        string
- Default:     "api.pcloud.com"
- Examples:
  - "api.pcloud.com"
    - Original/US region
  - "eapi.pcloud.com"
    - EU region

#### --pcloud-username

Your pcloud username.
			
This is only required when you want to use the cleanup command. Due to a bug
in the pcloud API the required API does not support OAuth authentication so
we have to rely on user password authentication for it.

Properties:

- Config:      username
- Env Var:     ZCLONE_PCLOUD_USERNAME
- Type:        string
- Required:    false

#### --pcloud-password

Your pcloud password.

**NB** Input to this must be obscured - see [zclone obscure](/commands/zclone_obscure/).

Properties:

- Config:      password
- Env Var:     ZCLONE_PCLOUD_PASSWORD
- Type:        string
- Required:    false

#### --pcloud-description

Description of the remote.

Properties:

- Config:      description
- Env Var:     ZCLONE_PCLOUD_DESCRIPTION
- Type:        string
- Required:    false

<!-- autogenerated options stop -->
