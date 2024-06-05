### Build
go build -o gses .


should be built statically
### Use

- dont forget the full path to the binary

```
# passing as flags
$ ./gses --secret-name pathtoajsonobjectofsecretenvs --project my-lovely-project -- /usr/bin/env
# passing as env vars:
$ export SECRET_NAME=pathtoajsonobjectofsecretenvs
$ export PROJECT=my-lovely-project
$ ./gses -- /usr/bin/env
```

short flags are supported:
```
$ ./gses -n pathtoajsonobjectofsecretenvs -p my-lovely-project -- /usr/bin/env
...
```

verbose prints the ENV VARS (use with caution)
```
$ ./gses -v -n pathtoajsonobjectofsecretenvs -p my-lovely-project -- /usr/bin/env
## Found 2 ENV VARs in the secret:
## GREETING=hello
## NAME=bob
...
```