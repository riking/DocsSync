# DocsSync

This program will download collaboratively edited files from Google Drive to your local computer. (Note: Google Drive is considered the source of truth; upload is not implemented)

## Build

    $ go get github.com/riking/DocsSync

    # Add to PATH (optional)
    $ ln -s ~/bin/DocsSync $GOPATH/bin/DocsSync
    $ sudo ln -s /usr/local/bin/DocsSync $GOPATH/bin/DocsSync

## Usage

To use, create a sync\_config.json file listing the local filenames and the Google Drive file IDs of the files you want to keep on your filesystem, and copy the `DocsSync` binary into the folder, then run as ./DocsSync.

TODO: Currently, you are required to set up your own https://console.developers.google.com project , because I don't want to be punished if someone goes and hits the quota. The key shown in the Git history is not valid, don't bother.

Select "API & Auth" -> "Credentials" -> "New OAuth Client ID", choose "Installed application", with a type of "Other".

Example:

```
you@desktop:~/Documents$ cat sync_config.json
{
  "client_id": "317xxxxxxxx-xxxxxxxxxxxxxxxxxx.apps.googleusercontent.com",
  "client_secret": "XxxxxXxxxXXXXxx-xxXX",
  "directory": "/home/you/Documents/",
  "files": [
    {
      "filename": "Taxes/2014.rtf",
      "file_id": "18vxbNiXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX2exhY",
      "mime": "application/rtf"
    },
    {
      "filename": "someCode/foo.go",
      "file_id": "1mDtDe7XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXqMzy8",
      "mime": "text/plain"
    }
  ]
}
you@desktop:~/Documents$ ./DocsSync
2015/04/08 17:35:30 Downloading Taxes/2014.rtf from 18vxbNiXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX2exhY
2015/04/08 17:35:30 Downloading someCode/foo.go from 1mDtDe7XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXqMzy8
2015/04/08 17:35:30 OK  Taxes/2014.rtf
2015/04/08 17:35:31 OK  someCode/foo.go
```

The program will create a .rftoken file. **This file gives full (read) access** to your Google Drive account. Don't lose it! The file is created with 0600 file permissions, but try not to include it in your backups.

### Feature Wishlist

 - What to do for the OAuth client secret for a 'desktop application'?
 - Check the modtime of the local and Drive files to only download if necessary
 - Pick a better name
 - Allow you to drag & drop the folder onto the binary or pick a folder with the command line
 - Allow for entering the OAuth authorization in a GUI instead of only terminal


