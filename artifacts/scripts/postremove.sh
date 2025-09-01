#!/bin/sh

echo "Executing postremove script"

if ! [ -e /home/dv/processing/dv-processing ]
 then
   echo "Dv Processing removed. Disabling..."
   if systemctl list-unit-files | grep "dv-processing.service"
    then
       systemctl disable dv-processing.service
       systemctl stop dv-processing.service
   fi
fi

echo "Postremove script done"
