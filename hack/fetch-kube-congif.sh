#!/bin/bash -e

: ${KEY_FILE?= required}
: ${PROJECT?= required}
: ${ZONE?= required}

gcloud auth activate-service-account --key-file=$KEY_FILE
gcloud config set project $PROJECT
gcloud --quiet container clusters get-credentials eng --zone $ZONE
