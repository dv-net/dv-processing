#!/bin/sh

echo "Executing postinstall script"


dv_user="${DV_USERNAME}"

if [ -z $dv_user ]; then
  dv_user="dv"
fi

id -u $dv_user || useradd $dv_user

usermod -aG $dv_user dv

if [ -e /home/dv/environment/processing.config.yaml  ] && ! [ -e /home/dv/processing/config.yaml ]
 then
   echo "Found dv-environment config. Copying..."
   cp /home/dv/environment/processing.config.yaml /home/dv/processing/config.yaml
fi

if [ -e /home/dv/processing/dv-processing.service  ] && ! [ -e /etc/systemd/system/dv-processing.service ]
 then
   echo "Unit file not exists. Copying..."
   cp /home/dv/processing/dv-processing.service /etc/systemd/system/dv-processing.service
fi

systemctl enable dv-processing.service
systemctl restart dv-processing.service

echo "Postinstall scripts done"