# Load Generator Setup
To generate a load inside kubernetes cluster, we need to follow three steps.

# Step 1: Reading The Number of Requests (DATA_NAME.sh)
The aim of using load generator image is to setup the php-apache development environment related to PHP file based on web apps. The first step to generate a load inside kubernetes is creating two shell script to read the records data from NASA-DAY.csv, FIFA-DAY.csv, RandomFIFA.csv, or NASA_Repeat.csv. In the <Data_Name.sh> script, we used the read command to read a line from the CSV file and sleep 6 or 60 seconds to read the next line.

# Step 2: Computing Load (index.php)
This step defines a way to simulate load in kubernetes cluster to generate some CPU intensive computations. The first line ini-set(’max-execution-time’, 600) sets the maximum time a script can run before it is terminated. The reason for adding this line is to prevent tying up the server. Then, the code gets the number of bytes from the previous step, and squares root the bytes.

# Step 3: Importing Load Generator (Dockerfile)
Image Dockerfile is the final step to insert set of instructions to create a docker image inside kubernetes. The content of Dockerfile shows below. The first line of Dockerfile is FROM jtkruthik99/loadgenerator:latest is eventually calls docker pull command from an existing docker image. Then, the next four lines are used to add the datasets, NASA and FIFA, from CSV file and shell script that reads the number of request from the CSV file. The following four lines are running a permission to read the added files.

# Buidl Dockerfile through three commands
Now, all the instructions have been set, so it is the time to build the docker image by using -t switch to set the tag of jtkruthik99/loadgenerator:latest. 
``` 
docker build -t jtkruthik99/loadgenerator:latest .
``` 
```
docker tag jtkruthik99/loadgenerator:latest loadgenerator:latest
```
```
sudo docker push jtkruthik99/loadgenerator:latest
```

# Run a load. 
To Run a specific dataset, you should run the following command that will build a pod called load-generator, and then read from a specific dataset to compute a load. 
```
kubectl run -i --tty load-generator --rm --image=jtkruthik99/loadgenerator:latest --restart=Never -- /bin/sh <DATA_NAME.sh>
```

# Note
1. If you change or edit any of the three file, you should rebuild dockerfile. 
2. If you load-generator pod is running, you can stop it by delete the pod or (CTRL-C)
3. Becareful of the dataset style as it might provide you an error if you use another format. 
4. 123nag is the docker account that we used while implementing this thesis, you can create your own account from (https://docker.io/)


