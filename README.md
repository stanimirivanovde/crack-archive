# RAR and ZIP password cracker

This password cracker is written in golang and attempts a brute force attack on the provided archive.
Currently up to 3 letter passwords are supported. The code is not optimize to be performent outside
of using multiple CPUs.

This is a draft project.

## Usage

```text
go run cmd/crack-archive.go -file ./test.rar
```

This will attempt to crack the test archive.

The project depends on `golift.io/xtractr` to open the archive with a password.
