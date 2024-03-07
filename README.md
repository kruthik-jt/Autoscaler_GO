# PodMetrics
Podemtrics interaction with metric server. Podemterics clean the data and puts in a TimeSeries DB. Grafana is used for visualization.

These are proposed autoscaling algorithms that are considering CPU metrics and historical data. All the algorithms written in go language, and follow the concept of defult Kubernetes autoscaling (Horizontal Pod Autoscaler).  

# Algorithm 1: One-step history
It is a simple algorithm as it takes only one previous decision into account when the algorithm calculates the diesred number of containers/replicas for the next time interval. The goal of taking one-step history is to reduce high fluctuation by agressively or lightly scaling up/down. The scaling period that we used for this algorithm is every 2 minutes. This algorithm has two conditions as follow. 

The first condision is when the system must scale up (the current utilization is greater than the target). The algorithm will check the previsou decision if it was up, te algorithm scale up more vigorously using HPA formula. However, if the previous decision is down, the algorithm lightly scale up by adding one more container/replica

The second condision is when the system must scale down (the current utlization is less than the target). The algorithm will check the previsou decision if it was up, te algorithm scale down ligly by decreasing one container/replica. However, if the previous decision is down, the algorithm scale down more vigorously using HPA formul

# Algorithm 2: Rolling Averages
This algorithm is using a rolling average of 5 previous measurements. Recall that the rolling average of a quantity X is a metric that utilizes the past L measurements Mi−L ,Mi−L+1 ,..., Mi−1, in order to calculate the average at time i. The goal of this averaging, acting as a low-pass filter, is smooth out the fluctiations in the workload. The scaling period is every 1 minute. This means that evey 1 minute the system will store one new measurements and shifting/removing the latest measurements (FIFO). Then, the result of the rolling average will be used for current utilization which is apply in HPA formula. 

# Algorithm 3: Moving Window Averages
This algorithm is a slight variation of the previous one; instead of rolling, it uses moving window averages. This means that the difference with the previous algorithm is that with moving averages, we “forget” the history before the current window. The goal of this algorithm is to have a low-pass effect. The scaling period for this algorithm is 5 minutes. This means that the system make a new decision at 5, 10, 15, ... minutes. 

# Algorithm 4: Horizontal Pod Autoscaler (HPA)
This algorithm is the orginal Kubernetes autoscaling algorithm that using the HPA formula. In this thesis, we creates a HPA for an existing Deployment called php-apache inside Kubernetes. HPA autoscaler maintain the target CPU utilization = 50%. The HPA scale up and down the replica counts between minimum = 1 and maximum = 10. We used two HPA cooling down settings. The fist setting is the default, which is the cooling down is set to 5 minutes while the other was set to be no cooling down (zero minutes). 

# Running Instructions for the Proposed Autoscaling
1. Git or copy a specific code (e.g. RollingAverage.go)
2. Compile the code 
``` 
go build
```
3. Run the code 
```
./fileName MinReplica
e.g. ./RollingAverage 1
```

In step 3, Each program will print the current container/replica, current CPU utilization [%], desired container/replica, provisioning acurracy metrics, and provisioing timeshare metrics every scaling period. To stop the program **(ctril-C)**

# Running Instructions for HPA in Kubernetes
1. Git or copy the yaml file (e.g. HPA_NoCooling.yaml)
2. Create the HPA autoscaler by running the following command
```
kubectl create -f <HPA_File_Name.yaml> [Use php-apache.yaml]
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
This command prints the current status and a description of everything. You can see the current and desired Utilization in milicore and precentage as well.  
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
3. Becareful to not run two autoscalers at the same time. 
