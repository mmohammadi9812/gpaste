# GPaste

Gpaste is a simple pastebin service, which can store images and texts to share with other users.

## Usage

```bash
docker-compose up -d
```

or, if you want to run without docker, make sure that redis, cassandra and minio are properly set up, then run:

```bash
# assuming you have redis, cassandra and minio up & running
go build .
./gpaste
```

## License

Copyright 2023 Mohammad Mohamamdi. All rights reserved. Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.
