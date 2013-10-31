[![Build Status](https://travis-ci.org/Altonymous/gopherswithgrenades.png)](https://travis-ci.org/altonymous/gopherswithgrenades)

# Gophers with Grenades 

This is a load testing suite that uses AWS and apachebench to slam your chosen URL

## Dependencies

# AWS account.
# PEM Key file to place on the server to manage connections
  - Must be in your ~/.ssh directory.
  - Must set the permissions to read-only.

## Local Build and Test

You can use go get command: 

    go get github.com/Altonymous/gopherswithgrenades 

Testing: (Not setup yet)

    go test github.com/Altonymous/gopherswithgrenades 


## Travis-CI

You can see a build status: https://travis-ci.org/Altonymous/gopherswithgrenades


## References

Building a Go Project: http://about.travis-ci.org/docs/user/languages/go/

How to Write Go Code: http://golang.org/doc/code.html

## Special Thanks
beeswithmachineguns - Inspiration and something to model my project after. (https://github.com/newsapps/beeswithmachineguns/)

IRC Freenode Channel - #go-nuts - For putting up with all my silly questions!
