### Build linux/amd64 binary
```shell
$ docker build -t gses:latest . && docker run -v $PWD/build/:/build -it gses:latest cp -rp /usr/local/bin/gses /build/
```

### Build / run for local platform
```shell
$ go build -o gses . # builds
$ go run . -- /usr/bin/env # builds and runs (executing /usr/bin/env)
```
should be built statically for production
### Use

- dont forget the full path to the binary as gses doesnt load PATH

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