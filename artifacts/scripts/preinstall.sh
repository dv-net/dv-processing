#!/bin/sh

echo "Executing preinstall script"

if systemctl list-unit-files | grep "dv-processing.service"
 then
   systemctl stop dv-processing.service
fi

echo "Preinstall script done"