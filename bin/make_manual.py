#!/usr/bin/env python3
"""
Make single page versions of the documentation for release and
conversion into man pages etc.
"""

import os
import re
import time
import subprocess
from datetime import datetime, timezone

docpath = "docs/content"
outfile = "MANUAL.md"

# Order to add docs segments to make outfile
docs = [
    "_index.md",
    "install.md",
    "docs.md",
    "remote_setup.md",
    "filtering.md",
    "gui.md",
    "rc.md",
    "overview.md",
    "tiers.md",
    "flags.md",
    "docker.md",
    "bisync.md",
    "release_signing.md",

    # Keep these alphabetical by full name
    "fichier.md",
    "alias.md",
    "s3.md",
    "archive.md",
    "b2.md",
    "box.md",
    "cache.md",
    "chunker.md",
    "cloudinary.md",
    "sharefile.md",
    "crypt.md",
    "compress.md",
    "combine.md",
    "doi.md",
    "drime.md",
    "dropbox.md",
    "filefabric.md",
    "filelu.md",
    "filen.md",
    "filescom.md",
    "ftp.md",
    "gofile.md",
    "googlecloudstorage.md",
    "drive.md",
    "googlephotos.md",
    "hasher.md",
    "huaweidrive.md",
    "hdfs.md",
    "hidrive.md",
    "http.md",
    "imagekit.md",
    "iclouddrive.md",
    "internetarchive.md",
    "internxt.md",
    "jottacloud.md",
    "koofr.md",
    "linkbox.md",
    "mailru.md",
    "mega.md",
    "memory.md",
    "netstorage.md",
    "azureblob.md",
    "azurefiles.md",
    "onedrive.md",
    "opendrive.md",
    "oracleobjectstorage/_index.md",
    "qingstor.md",
    "quatrix.md",
    "sia.md",
    "swift.md",
    "pcloud.md",
    "pikpak.md",
    "pixeldrain.md",
    "premiumizeme.md",
    "protondrive.md",
    "putio.md",
    "protondrive.md",
    "seafile.md",
    "sftp.md",
    "shade.md",
    "smb.md",
    "storj.md",
    "sugarsync.md",
    "ulozto.md",
    "union.md",
    "webdav.md",
    "yandex.md",
    "zoho.md",

    "local.md",
    "changelog.md",
    "bugs.md",
    "faq.md",
    "licence.md",
    "authors.md",
    "contact.md",
]

# Order to put the commands in - any not on here will be in sorted order
commands_order = [
    "zclone_config.md",
    "zclone_copy.md",
    "zclone_sync.md",
    "zclone_move.md",
    "zclone_delete.md",
    "zclone_purge.md",
    "zclone_mkdir.md",
    "zclone_rmdir.md",
    "zclone_check.md",
    "zclone_ls.md",
    "zclone_lsd.md",
    "zclone_lsl.md",
    "zclone_md5sum.md",
    "zclone_sha1sum.md",
    "zclone_size.md",
    "zclone_version.md",
    "zclone_cleanup.md",
    "zclone_dedupe.md",
]    

# Docs which aren't made into outfile
ignore_docs = [
    "downloads.md",
    "privacy.md",
    "sponsor.md",
    "amazonclouddrive.md",
    "backends.md",              # Makes JSON confusingly
]

def read_doc(doc):
    """Read file as a string"""
    path = os.path.join(docpath, doc)
    with open(path) as fd:
        contents = fd.read()
    parts = contents.split("---\n", 2)
    if len(parts) != 3:
        raise ValueError(f"{doc}: Couldn't find --- markers: found {len(parts)} parts")
    contents = parts[2].strip()+"\n\n"
    # Remove icons
    contents = re.sub(r'<i class="fa.*?</i>\s*', "", contents)
    # Interpret img shortcodes
    # {{< img ... >}}
    contents = re.sub(r'\{\{<\s*img\s+(.*?)>\}\}', r"<img \1>", contents)
    # Make any img tags absolute
    contents = re.sub(r'(<img.*?src=")/', r"\1//", contents)
    # Make [...](/links/) absolute
    contents = re.sub(r'\]\((\/.*?\/(#.*)?)\)', r"](/\1)", contents)
    # Add additional links on the front page
    contents = re.sub(r'<!-- MAINPAGELINK -->', "- [Donate.](//donate/)", contents)
    # Interpret provider shortcode
    # {{< provider name="Amazon S3" home="https://aws.amazon.com/s3/" config="/s3/" >}}
    contents = re.sub(r'\{\{<\s*provider.*?name="(.*?)".*?>\}\}', r"- \1", contents)
    # Remove remaining shortcodes
    contents = re.sub(r'\{\{<.*?>\}\}', r"", contents)
    contents = re.sub(r'\{\{%.*?%\}\}', r"", contents)
    return contents

def check_docs(docpath):
    """Check all the docs are in docpath"""
    files = set(f for f in os.listdir(docpath) if f.endswith(".md"))
    files.update(f for f in docs if os.path.exists(os.path.join(docpath,f)))
    files -= set(ignore_docs)
    docs_set = set(docs)
    if files == docs_set:
        return
    print("Files on disk but not in docs variable: %s" % ", ".join(files - docs_set))
    print("Files in docs variable but not on disk: %s" % ", ".join(docs_set - files))
    raise ValueError("Missing files")

def read_command(command):
    doc = read_doc("commands/"+command)
    doc = re.sub(r"### Options inherited from parent commands.*$", "", doc, 0, re.S)
    doc = doc.strip()+"\n"
    return doc

def read_commands(docpath):
    """Reads the commands an makes them into a single page"""
    files = set(f for f in os.listdir(docpath + "/commands") if f.endswith(".md"))
    docs = []
    for command in commands_order:
        docs.append(read_command(command))
        files.remove(command)
    for command in sorted(files):
        if command != "zclone.md":
            docs.append(read_command(command))
    return "\n".join(docs)

def main():
    check_docs(docpath)
    command_docs = read_commands(docpath).replace("\\", "\\\\") # escape \ so we can use command_docs in re.sub
    build_date = datetime.fromtimestamp(
            int(os.environ.get('SOURCE_DATE_EPOCH', time.time())), timezone.utc)
    help_output = subprocess.check_output(["build/zclone", "help"]).decode("utf-8")
    with open(outfile, "w") as out:
        out.write("""\
%% zclone(1) User Manual
%% Nick Craig-Wood
%% %s

# NAME

zclone - manage files on cloud storage

# SYNOPSIS

```
%s
```
""" % (build_date.strftime("%b %d, %Y"), help_output))
        for doc in docs:
            contents = read_doc(doc)
            # Substitute the commands into doc.md
            if doc == "docs.md":
                contents = re.sub(r"The main zclone commands.*?for the full list.", command_docs, contents, 0, re.S)
            out.write(contents)
    print("Written '%s'" % outfile)

if __name__ == "__main__":
    main()
