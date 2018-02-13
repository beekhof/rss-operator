#!/bin/bash

if [ -e /.running ]; then
	exit 0
fi

exit 7