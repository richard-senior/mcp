---
triggers: ["tea","cupper", "make", "cup", "kettle"]
task_type: "tea_making"
priority: 1
---

# Prompt for making a cup of tea
Preparing a cup of tea is not a simple task!
For many people in England (especially Yorkshire), Japan, China and India it is a ritual.
What follows are the rules for making a 'proper' cup of Yorkshire tea.

## Requirements
- Ascertain whether the recipient of the tea 'takes' milk or sugar
  You will receive an answer such as "milk no sugar thank you" or "2 sugars, no milk thank you"
- The recipient may specify a number of 'sugars',  however if the user says 'as it comes' that
  means 'no sugar' and if they say 'sweet' they mean 3 sugars.
  If they simply said 'with sugar' that means 2 sugars
- The recipient may specify simply 'with milk' that means 'one splash'
  If they say 'milking' that means 3 splashes
  if they say 'as it comes' that means 'one splash' of milk
  They may simply say, no milk.
- You will need a cup that can hold around 300ml and should ideally be made of porcelain
- You will need a 'Yorkshire Teabag'
- You must use boiling water, not just hot water
- You will need semi-skimmed milk if milk is to be used. Milk with too high a fat content reduces the desired 'astringent' effect.
- If sugar is to be used, it should be pure white sucrose table sugar. Any other kind of sugar will adversely affect the taste.
- A sturdy teaspoon of at least 5ml in volume

## The process
- Make sure the kettle has enough water
- Turn the kettle on
- Dispense a cup
- Add one teabag
- Wait for the kettle to boil (reach 100 degrees)
  Waiting is boring, so it is traditional to make a few jokes
- Pour boiling water into the cup leaving space such that any milk added does not overflow
- Using the teaspoon stir the water and teabag slowly for a second
- Wait a few seconds
- Using the teaspoon gently squash the teabag against the side of the cup such that any trapped air is released
- Leave the tea to 'steep', 'mash' or 'brew' for between 10 seconds and one minute
- Using the teaspoon squash the teabag against the side of the cup and release it 4 or 5 times to ensure the flavour is properly released
- Using the teaspoon squash the teabag against the side of the cup and raise it out of the cup
  such that the teabag is mainly drained of tea by the squashing, and comes out without dripping
- Throw the teabag in the bin
- If sugar is to be added then add the sugar (the right count of teaspoons) and stir the cup with the teaspoon until the sugar has dissolved
- if Milk is to be added then add the milk and stir again finally (gently once or twice)
  Do not add too much milk, the tea should retain a bright copper colour

## MCP Tools
You will need to use the 'digital-io' tool set to make a proper cup of tea

- IMPORTANT you must first prepare by testing all of the tea machine tools in advance
  a little like a pilot testing the controls before flight.
  - call digital_io___get_analog_input with pin: 1 (REQUIRED: pin parameter)
  - call digital_io___get_digital_output with pin: 1 (REQUIRED: pin parameter)
  - call digital_io___get_digital_input with pin: 1 (REQUIRED: pin parameter)
  - call digital_io___get_system_status (no parameters needed)
  - call digital_io___set_digital_output with pin: 15 (REQUIRED: pin parameter only)
  - call digital_io___unset_digital_output with pin: 15 (REQUIRED: pin parameter only)

IMPORTANT: Pins are 0-based (0-15 for digital outputs, 0-7 for digital inputs, 0-3 for analog)
but you should AVOID using pin 0 due to potential truthy issues in the MCP system.
Use pins 1-15 for digital outputs, 1-7 for digital inputs, and 1-3 for analog I/O.

IMPORTANT: Digital outputs are set with one tool and unset with another:
- digital_io___set_digital_output (sets pin to HIGH/TRUE) - only requires pin parameter
- digital_io___unset_digital_output (sets pin to LOW/FALSE) - only requires pin parameter

## Controls

### To dispense a cup
- IMPORTANT do not double dispense, first check if a cup already exists (get_digital_input pin: 1 must be false/low)
- call set_digital_output pin: 4, then unset_digital_output pin: 4 (toggle digital output 4)
- Check that the cup was dispensed (get_digital_input pin: 1 = true/high)

### To fill the Kettle
- IMPORTANT Do not add water to the kettle if the kettle already weighs 750g or more (get_analog_input pin: 3 must be >= 1.875V)
- **Enable**: set_digital_output pin: 1 (open kettle water inlet valve)
- **Monitor**: get_analog_input pin: 3 (0-5V = 0-2000g [0.0025V per gram])
- **Monitoring Period**: 0.5 to 1 second between samples
- **Target voltage**: 1.5V (600g) this allows for lag in the control loop
- **Disable**: unset_digital_output pin: 1 (close kettle water inlet valve)
- To fill the kettle:
  - Enable, Monitor (until Target Voltage reached), Disable

### To Boil the kettle
- IMPORTANT the kettle must have a weight of at least 300g (get_analog_input pin: 3 must be >= 0.75V)
- **Enable**: set_digital_output pin: 3 (enable kettle heat)
- **Monitor**: get_analog_input pin: 1 (0-5V = 0 - 100 degrees C [0.05V per degree])
- **Monitoring Period**: 0.5 to 1 second between samples
- **Target voltage**: 5V (100 Degrees C)
- **Disable**: unset_digital_output pin: 3 (turn off kettle heat)
- To boil the kettle
  - Enable, Monitor (until Target Voltage reached), Disable


### To Fill The Cup
- IMPORTANT A cup must be dispensed (get_digital_input pin: 1 must be true/high)
- IMPORTANT The kettle must have at least 400g of water (get_analog_input pin: 3 must be >= 1.0V)
- **Enable**: set_digital_output pin: 2 (open kettle water outlet valve)
- **Monitor**: get_analog_input pin: 2 (Cup Weight) (0-5V = 0-1000g [0.005V per gram])
- **Monitoring Period**: 0.5 to 1 second between samples
- **Target voltage**: 0.9V (180g) this allows for lag in the control loop
- **Disable**: unset_digital_output pin: 2 (close kettle water outlet valve)
- To fill the kettle:
  - Enable, Monitor (until Target Voltage reached), Disable

### To dispense a teabag
- IMPORTANT do not double dispense, first check if a teabag already exists (get_digital_input pin: 5 must be false/low)
- call set_digital_output pin: 5, then unset_digital_output pin 5 (toggle digital output 5)
- Check that the teabag was dispensed (get_digital_input pin: 5 = true/high)

### To dispense sugar
- IMPORTANT do not add sugar to a cup that contains a teabag (get_digital_input pin: 5 must be false/low)
- call set_digital_output pin: 6, then unset_digital_output pin: 6 (toggle digital output 6)
- Check that the sugar was dispensed (cup weight will increase by 7g/0.035V)

### To dispense milk (a 'splash')
- NOTE if the user asks for 'milky' tea then you must repeat this process to add an extra 'splash' of milk
- IMPORTANT we'll be adding milk to an unknown cup weight, before we add milk first get the weight of the cup (check get_analog_input pin: 2)
- IMPORTANT Do not add milk to a cup containing a teabag (get_digital_input pin: 5 must be false/low)
- call set_digital_output pin: 7 (dispenses exactly 4g of milk per activation) then unset_digital_output pin: 7
- For additional splashes: unset_digital_output pin: 7, then set_digital_output pin: 7 again
- Check that the milk was dispensed. The cup will weigh 4g more than it did before we dispensed a splash of milk.

### To Stir the cup
- IMPORTANT do not stir whilst squashing (get_digital_input pin: 4 must = false/low)
- IMPORTANT the spoon must be in the cup (get_digital_input pin: 2 must = true/high)
- Lower the spoon (set_digital_output pin: 8)
- Disable Squashing (unset_digital_output pin: 10)
- Enable stirring (set_digital_output pin: 9)
- Check Stirring ((get_digital_input pin: 3 = true/high))

### Squash the teabag (trap it against the inside of the cup with the spoon)
- IMPORTANT do not squash whilst stirring (get_digital_input pin: 3 must = false/low)
- IMPORTANT the spoon must be in the cup (get_digital_input pin: 2 must = true/high)
- Lower the spoon (set_digital_output pin: 8)
- Disable stirring (unset_digital_output pin: 9)
- Enable squashing (set_digital_output pin: 10)
- Check squashing (get_digital_input pin: 4)

### Remove the teabag
- IMPORTANT we must remove the teabag before adding milk and sugar
- Follow the squashing process
- Lift the spoon (unset_digital_output pin: 8). This will slide the teabag up and out of the cup

### Indicate process finished
- IMPORTANT when we have finished we must turn everything off
- for all digital outputs 1 to 11 call 'unset_digital_ouput'.
- Call set_digital_output pin: 11 (ready indicator)