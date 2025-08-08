# Twist Auto-Login Script for Alien Retribution
# Based on TWX Script Pack 1: TWGS Login Script
# Modified for Alien Retribution login sequence

# Set your login credentials here:
LoginName = "mrdon"
Password = "bob"  
Game = "a"

# Terminate script if disconnected
setEventTrigger 0 :End "Connection lost"

# Wait for initial login prompt and send username
waitfor "(ENTER for none): "
send LoginName "*"

# Wait for game selection menu and select game
waitfor "Selection (? for menu):"
send Game

# Wait for and skip the rules/animation pause screen
waitfor "[Pause]"
send "*"

# Wait for game menu and select "T - Play Trade Wars 2002"
waitfor "Enter your choice"
send "t" "*"

# Wait for log prompt and skip it
waitfor "Show today's log?"
send "*"

# Wait for and skip another pause screen
waitfor "[Pause]"
send "*"

# Wait for password prompt and enter password
waitfor "Password"
send Password "*"

# Wait for final pause screen after login messages
waitfor "[Pause]"
send "*"

# Set up triggers for successful login and pause handling
setTextLineTrigger 1 :End "Sector  : "
setTextTrigger 2 :Pause "[Pause]"
setTextTrigger 3 :Pause "Do you wish to clear some avoids?"

# Pause to let triggers activate
pause

:Pause
# Handle pause prompts by sending enter
send "*"
killTrigger 2
setTextTrigger 2 :Pause "[Pause]"
pause

:End
send "/"
# Script ends when we reach the command prompt (Sector line) or disconnect