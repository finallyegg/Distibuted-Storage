# My storage

### Features

- Starts from simple key - value storage
- Dynamic chunk large file into small files
- Sync between two replicas
- Integrated with file system and support versioning with user defined interval [1] (use with macFuse)

<hr>
### Structure
 #### Client: 
  - ##### mounted on user file system 
  - ##### accept user command or action
  - ##### Communicate with server using htttp request to do CRUD operations

#### Server:
- ##### give response clients request
- ##### written data to database
- ##### saving version metadata in memory 

<hr>
### How to install? 

- ###### install macfuse 3.11.1
- ###### clone repo use either git clone / go get
- ###### put into your go PATH

- run:  ``go run server.go -s [port number]`` (because we are running in local host)
example: ``go run server.go -s 8888``

- run ``go run main.go`` with following arguement:
 - -m mount point (optional) by default it mounts 'dss' in your current dir
 - -s server address: location of your server
 - -f flush interval: how often should system save a version if changes are detected (optional)
 - -n start a new system (optional)
 - -t time travel: go back to a certain time if a copy exist:
 -example: ``-t "2015-09-15T12:40"``

<hr>

### Usage
- ##### CRUD operation:
 - perform any command from mkdir/touch to compile a large library like reddis

- ##### get a file versions:
 - ``mkdir filename@@versions``
 
 <hr>
 
### Demo:
- ##### Video #1: [https://youtu.be/yb23VdogB9U]
- ##### Video #2: [https://youtu.be/lFucu_b2oC4]




 


