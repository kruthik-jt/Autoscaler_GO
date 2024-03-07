#! /bin/sh
while IFS=, read -r reply; do
#echo "$reply"
reply=$(echo $reply | sed 's/,,,//')
echo "$reply"
wget -q -O- http://php-apache/index.php?bytes=$reply
sleep 60
done < RandomFIFA.csv
