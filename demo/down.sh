#!/bin/bash

set -e
set -x

c1=c1
c2=c2

kind delete cluster --name "${c1}"
kind delete cluster --name "${c2}"