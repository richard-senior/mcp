# MCP Project Helper

## Overview
This directory (/Users/richard/scripts/mcp) contains the 'MCP' project.
MCP is a go application into which I will place personal MCP tools for use by AmazonQ as and when I
discover a need for them

## Usage
Use this prompt when working on MCP project in combination with any other rules specifying general or  golang practices etc.

## Project Creation
We should create a GO application which implements an MCP "client" compatible with usage by Amazon Q.
The Client will *not* connect to an MCP server but rather it will fullfil the tool request directly.

That is to say, if there is a tool called Calculator which simply implements arithmetic operations then
the client should handle requests by AmazonQ, hand off to some go function which implements calculator, and then returns the results to Q.

Again, we are not creating an MCP server, but an MCP client which also implements tools directly after being invoked on the command line by AmazonQ.

## mcp features
The mcp application should feature an example tool (a simple calculator)
Tools should be implemented in a structural manner allowing for creation of new tools as simply as possible.

## General
All dates and spellings should be in British (UK) formats