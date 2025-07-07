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

# CSS Styling
SVG's can include a css style elements at the top of the file which may be used later in the file
to apply styles to svg elements:
```xml
<svg>
   <style>
        :root {
            font-family: Arial, sans-serif;
            fill: #333;
        }
        .title {
            font-size: 24px;
            font-weight: bold;
        }
        <!-- more css elements here -->
    </style>

    <!-- possible other SVG elements here -->
    <text x="600" y="40" class="title" text-anchor="middle">Lorem Ipsum</text>
    <!-- possible other SVG elements here -->
</svg>
```
In this example we can see that the 'title' style is used by the SVG <text> element
When creating SVG's we should use this mechanism in preference for any other method of styling SVG elements

# Scripting
Similarly to CSS, SVG's can contain script blocks that can be used for animation or user interaction etc.
Below is an example of using embedded javascript to create a 'drill down' mechanism whereby users can click on elements
of the SVG and have the SVG show a different SVG etc.
```xml
<svg width="1200" height="900" viewBox="0 0 1200 900" id="main" style="visibility: visible;">
    <script>
        function drillDown(elementId) {
            var m = document.getElementById('main');
            var t = document.getElementById(elementId);
            m.style.visibility = 'hidden';
            t.style.visibility = 'visible';
        }
    </script>
    <!-- possible other SVG elements here -->

    <!-- clicking the rect will 'drill down' -->
    <a onClick="drillDown('jsp_templates');">
        <rect x="150" y="150" width="180" height="80"/>

    </a>
    <!-- possible other SVG elements here -->
    <g>
       <svg width="1200" height="900" viewBox="0 0 1200 900" id="jsp_templates" style="visibility: hidden;">
            <!-- click to go back to main -->
            <a onClick="drillUp('jsp_templates');">
                <text x="100" y="70" class="back-link">← Back to Architecture Overview</text>
            </a>
    </g>
</svg>
```
If you are asked to create an SVG with 'drill down' functionality then this is the mechanism we should use
This mechanism (with different scripting) can be used for animation or any other possibilities

# Slides (Powerpoint)
We can use the scripting mechanism to create a portable (across OS's etc.) slide deck
If you're asked to make a slide deck or a 'powerpoint' you can do it like this:
```xml
<?xml version="1.0"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"  width="400" height="400" viewBox="0 0 400 400">
    <style>
        :root {
            font-family: Arial, sans-serif;
            fill: #333;
        }
        .main-text {
            font-size: 24px;
            font-weight: bold;
            text-anchor: middle;
        }
        .secondary-text {
            font-size: 20px;
            text-anchor: middle;
        }
        .box {
            fill: none;
            stroke: #333;
            stroke-width: 2;
            cursor: pointer;
        }
        .clickable {
            cursor: pointer;
        }
    </style>

    <script>
        <![CDATA[
        let currentStage = 0;
        let totalStages = 0;

        // Initialize by counting all stage elements
        function initStages() {
            let stageCount = 0;
            let element;
            while ((element = document.getElementById('stage' + stageCount)) !== null) {
                if (stageCount > 0) {
                    element.style.visibility = 'hidden';
                }
                stageCount++;
            }
            totalStages = stageCount;
        }

        function nextStage() {
            // Hide current stage
            document.getElementById('stage' + currentStage).style.visibility = 'hidden';

            // Advance to next stage
            currentStage = (currentStage + 1) % totalStages;

            // Show new current stage
            document.getElementById('stage' + currentStage).style.visibility = 'visible';
        }

        // Initialize when SVG loads
        document.addEventListener('DOMContentLoaded', initStages);
        // Fallback for SVG loaded directly
        setTimeout(initStages, 100);
        ]]>
    </script>

    <!-- Outer box that encompasses the entire SVG -->
    <rect x="10" y="10" width="380" height="380" class="box clickable" onclick="nextStage()"/>

    <!-- Stage 0 -->
    <g id="stage0" style="visibility: visible;">
        <text x="200" y="180" class="main-text">This is some text</text>
        <text x="200" y="220" class="secondary-text">More is to come</text>
    </g>

    <!-- Stage 1  -->
    <g id="stage1" style="visibility: hidden;">
        <text x="200" y="200" class="main-text">ok this is some more text</text>
    </g>
    <!-- etc. -->
    <!-- Invisible clickable area to ensure clicks work anywhere in the box -->
    <rect x="10" y="10" width="380" height="380" fill="transparent" class="clickable" onclick="nextStage()"/>
</svg>

```