# GRBL is a language for controlling CNC machines
In our case the machine is a three axis engraving machine where:
- The XY axis forms a plane which is parallel to the ground and the Z axis is perpendicular to that plane
- The Z axis holds the tool or pen etc, and the X and Y axes move the work piece under the tool or pen

## Syntax
- A GCODE command is a letter followed by a number such as M00 or X-0.005 etc.
- a GRBL command is a sequence of GCODE$ commands grouped on a single line of a text file
- A GRBL script/file is a sequential list of GRBL commands each on a separate line
- The machine begins processing the file at the top executing the commands until there are no more
- Only things that have changed need stating ie. if the feed rate is '10' it will remain 10 until changed.

- Some example GRBL commands are are :
  G00 Z5.000000 (Rapid move the Z axis to 5 units above the Z-Axis zero point)
  M3 (Start spindle clockwise)
  G00 X138.620502 Y-113.213650 (Rapid move the X and Y axis to the given coordinates)
  G01 Z0.000000 F100.0 (Move the Z Axis to position zero at feed rate 100)

- Generally GCODE commands are organised into "blocks" where each block represents a single 'path' or 'cut'.
  So a block will begin by lowering the Z axis to 'cutting depth', moving the X and Y axes to perform the cut
  and then raising the Z axis at the end of the cut

Here is a list of the most used GCODE's
- F (Feed Rate, the speed at which the next move will happen ie. F200)
- G00 (Move at 'Rapid' speed)
- G01 (Move at cutting speed)
- G02 (Cut a clockwise arc. Must be followed by I, K or J)
- G03 (Cut an anticlockwise arc. Must be followed by I, K or J)
- G17 (select the XY plane for arcs. Default)
- G18 (select the XZ plane for arcs)
- G19 (select the YZ plane for arcs)
- G90 (Absolute Positioning, default on most machines)
- G21 (Units are metric ie mm)
- M02 (Stop all)
- M03 (Start the spindle clockwise)
- M05 (Stop the spindle)
- S (Spindle RPM ie S10000)
- X (A position on the X axis ie X25.30)
- Y (A position on the Y axis ie U-30.6)
- Z (A position on the Z axis ie Z-1)
- I (Distance in the X Axis from the current X coordinate to the centre of an arc circle for G02 and G03)
- J (Distance in the Y Axis from the current Y coordinate to the centre of an arc circle for G02 and G03)
- K (Distance in the Z Axis from the current Z coordinate to the centre of an arc circle for G02 and G03)
- R (Radius for use in G02 and G03 commands ie R2.0)

## Example
Say we wish to draw a square 10mm wide on the XY axis
We would start by putting some paper on the XY plane lower the z axis until the pen is just touching the paper.
This is our Z0 position. We then manually raise the pen by 1mm, and manually move the X and Y axis to our zero position.
Then we do:
```grbl
G21 (All units are metric)
G00 X10.00 Y10.00 F800 (move at 800mm/min to 10,10, our starting coordinate)
G01 Z0 F5 (lower the pen slowly (F5) to be touching the paper)
G01 X10.00 Y20.00 F200 (Quickly draw a vertical line upwards at 200mm/min)
G01 X20.00 Y20.00 (Draw a horizontal line using the current speed (200mm/min))
G01 X20.00 Y10.00 (Draw a vertical line downwards)
G01 X10.00 Y10.00 (Draw back to the start to finish the square)
G00 Z0 F800 (lift the pen to zero)
G00 X0.00 Y0.00 (move back to origin)
```

# Arcs (And Bezier Curves)
## Arcs with the IJK method
```grbl
G00 X0 Y0 (Move to the start of the arc at X=0 Y=0)
G02 X2 Y2 I1 J1 (Draw the arc with centre point I=1, J=1 [X+1, Y+1])
```
This results in a semicircular arc starting at 0,0 and ending at 2,2 passing through 0,2

## Arcs with the R Method (poorly supported)
```grbl
G00 X0 Y0 (Move to the start of the arc at X=0 Y=0)
G02 X2 Y2 R1.4142 (Draw the arc with centre point I=1, J=1 [X+1, Y+1])
```
This results in a semicircular arc starting at 0,0 and ending at 2,2 passing through 0,2
The radius is the square root of two (basic trig)


## General GRBL rules
- The machine is only accurate to 0.1mm so all floating point numbers can be restricted to 3 decimal places
- Z0 is always a safe distance ABOVE the XY plane
- Scripts must start with
- G21 (metric)
- G90 (Absolute Positioning)
- paths/blocks of commands must start with Z going down and end with Z going back up
- paths/blocks must have no blank lines between Z down and Z up
- There must be at least one blank line between paths (blocks of GRBL commands)
