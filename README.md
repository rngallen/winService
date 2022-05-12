# winService
Run any Go executable file as a windows service

- [ ] There are two options on how to create windows services
  - [x] Hardcode everything within source code as in [main.go](/main.go)
  - [x] Using [json](/winService.json) file for configuration as [withEnv.md](/withEnv.md)

## Option A ( Everthing on SoureCode)
- [ ] Compile your program `go build .` then place compiled program on the same folder/directory with your go lang program.
- [ ] Open command line as administrator
- [ ] Run `winservice.exe -service install` 

## Option B (Using json file for configuration)
- [ ] Copy everything from [withEnv.md](/withEnv.md) and replace on [main.go](/main.go)
- [ ] Modify config file [json](/winService.json)
- [ ] Compile your program `go build .` then place it on the same directory with your config file [json](/winService.json)
- [ ] Your actual program can be on any directory but specify on config file
- [ ] Open command line as administrator
- [ ] Run `winservice.exe -service install` 

