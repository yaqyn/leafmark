#!/bin/sh
set -eu
printf '%s\n' 'Leafmark now applies additive migrations transactionally at startup.' 'Start the server with your DATABASE_URL configured. Back up the database before upgrading.'
