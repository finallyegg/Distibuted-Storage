# Project 1


Allow three operations

  - put
  - get
  - desc

# Put file
 #### [usage] ```put <file path>```
 #### [output] ```print top level reciept```
  - tear down a file with each chunk <= 8192 bytes
  - generate hash for both reciept and file block
  - print top - level reciept

# get file
 #### [usage] ```get <sig> newPath```
 #### [output] ```nil```
 - restore file from hashStore
 
# desc file
 #### [usage] ```desc <sig> ```
 #### [output] ```print reciept json string to stderr``` OR   ```print file size to stdout```



 - restore file
