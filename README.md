# PodMetrics
Pod metrics interaction with metric server. Pod metrics clean the data and puts in a TimeSeries DB. Grafana is used for visualization.

These are proposed autoscaling algorithms that are considering CPU metrics and historical data. All the algorithms written in go language, and follow the concept of defult Kubernetes autoscaling (Horizontal Pod Autoscaler).  
 
# Running Instructions for the PID Algorithm
1. Git or copy the specific code (e.g. PIDAlgo.go)
2. Compile the code 
``` 
go build
```
3. Run the code 
```
./fileName MinReplica
e.g. ./PIDAlgo 1
```

In step 3, Each program will print the current container/replica, current CPU utilization [%], desired container/replica, provisioning acurracy metrics, and provisioing timeshare metrics every scaling period. To stop the program **(ctril-C)**

# Running Instructions for HPA in Kubernetes
1. Git or copy the yaml file (e.g. HPA_NoCooling.yaml)
2. Create the HPA autoscaler by running the following command
```
kubectl create -f <HPA_File_Name.yaml>
kubectl create -f php-apache.yaml
```
4. To check the current container/replica, current CPU utilization [%], and desired container/replica, you can run one of these commands. 
This command prints the current status only once. 
```
kubectl get hpa
```
This command will watch any change happen and provide an update once it happen; however, it stop and you should rerun the command around 25-30 minuts. 
```
kubectl get hpa cpu-autoscale --watch
```
This command prints the current status and a description of everything. You can see the current and desired Utilization in milicore and percentage as well.  
```
kubectl describe hpa
```

**Note:**
1. You can check the number of current replica 
```
kubectl get pods
```

2. You can resize the current replica: 
```
kubectl scale deployment/php-apache --replicas=(set the number you want) [This can be used while the deployment is already running]
```
3. Be careful to not run two autoscalers at the same time. 
