# SVG is a vector graphics format
SVG is an XML tag based syntax defined as:
```xml
<?xml version="1.0"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
```
For our purpose we are interested in a small subsection of the xml definition relating to
paths and scaling etc

# Paths

An example path tag in an SVG file looks like this:
```svg
<path id="lineAB" d="M 100 350 l 150 -300" stroke="red" stroke-width="4" />
```
Here we are interested only in the 'id' and 'd' attributes of the path
## id
The id is the name of the shape that the path defines, as determined by the user or software which created the SVG file.
## d
The d attribute contains the actual geometry of the path in the form of a list of 'path commands'
### Path Commands
Like GRBL/GCODE a path command is a letter followed by a number.
Generally a capital letter deals with ABSOLUTE coordinates and a lower case letter deals with
coordinates RELATIVE to the current point.
The numbers (coordinates etc.) Can either be integers or floating point and can be negative or zero
#### Move Commands (like G00 in GRBL/GCODE)
- M (Move To absolute X, Y: Pn = {x, y}) ie. M 10,10
- m (Move relative to the current position, dX, dY: Pn = {xo + dx, yo + dy}) ie. m 10,10
#### Line To Commands (Like G01 in GRBL/GCODE)
- H (Draw a horizontal line (X axis) from the current position to the absolute X position: Po′ = Pn = {x, yo}) ie. H 5
- h (Draw a horizontal line (X axis) from the current position to a relative X position: Po′ = Pn = {xo + dx, yo}) ie. h 5
- V (Draw a vertical line (Y axis) from the current position to the absolute Y position: Po′ = Pn = {xo, y}) ie. V 5
- v (Draw a horizontal line (T axis) from the current position to a relative Y position: Po′ = Pn = {xo, yo + dy}) ie. v 5
- L (Draw a line from the current XY to the specified absolute XY: Po′ = Pn = {x, y}) ie. L 90,90
- l (Draw a line from the current XY to the specified relative XY: Po′ = Pn = {xo + dx, yo + dy}) ie. l 90,90
#### Quadratic Bezier Curves
- Q Draws a quadratic bezier curve from
    the current point to the ABSOLUTE end point, with the curve specified by the ABSOLUTE control point.
    For example:
        Q 25,25 40,50
    Draws a quadratic bezier from the current XY to 25,25 with a control point at 40,50
    The forumula of the curve drawn is : 𝐵(𝑡)=(1−𝑡)2𝑃0+2𝑡(1−𝑡)𝑃1+𝑡2𝑃2
- q (As Q but a destination and control point specified RELATIVE TO the current position)
- T and t: Another quadratic bezier system that we are not concerned with
#### Cubic Bezier Curves
- C, c, S, s are beyond our scope
#### Arcs (elliptical curves)
- A
    Draws an Arc (actually an ellipse) from the current point to the ABSOLUTE end point
    The exact specification of the arc is given by : (rx ry angle large-arc-flag sweep-flag x y)
    - rx and ry are the two radii of the ellipse
    - angle represents a rotation (degrees) of the ellipse relative to the x-axis
    - large-arc-flag and sweep-flag determine which of the 4 possible arcs to draw
    - x, y is the destination of the arc
    For example :
    ```text
    M 50,0 A 50,50 0 0 1 100,50
    ```
    Moves to 50,0 and draws a clockwise circular 90 degree arc ending at 100,50
#### Commands
- Z (Closes the current path by connecting the current point to the initial point using a straight line if necessary)
