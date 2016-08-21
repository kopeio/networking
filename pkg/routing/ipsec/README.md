## IPSEC


We normally run IPSEC with UDP encapsulation

UDP requires a user-mode listener (included).  It doesn't have to do anything.

We set up for each remote node:

AH & ESP policies in both direction
A policy that AH & ESP are required
