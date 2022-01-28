# dumpinen

Dumpinen is a library and cli tool that should be used together with the
dumpinen server, the cli supports UNIX-style composability.

The dumpinen server code is located here:
https://github.com/osm/dumpinen-server

## Example

```sh
# Store foo at the dumpinen server
$ echo foo | dumpinen
https://dumpinen.com/71m3YDlGNfC

# Fetch the contents we stored
$ dumpinen -id 71m3YDlGNfC
foo

# It's also possible to fetch with curl
$ curl https://dumpinen.com/71m3YDlGNfC
foo

# We can also choose to delte the file after 20 minutes
$ echo foo | dumpinen -delete-after 20m

# It's also possible to password protect the file
$ echo foo | dumpinen -cred foo:bar
https://dumpinen.com/rGNgSeSWeT0

# Fetching the dump without credentials will return an unauthorized error
$ dumpinen -id rGNgSeSWeT0
unauthorized

# But with the correct credentials we are allowd to fetch the file
$ dumpinen -id rGNgSeSWeT0 -cred foo:bar
foo

# And as always we can also use curl
$ curl -u foo:bar https://dumpinen.com/rGNgSeSWeT0
foo
```
