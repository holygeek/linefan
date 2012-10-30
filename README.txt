linefan - Spinning fan progress meter (a poor, half-assed mimickcry of pv).

I made this to show a spinning fan "progress" indicator when running a lengthy
"make" session.

Then I wondered "How long is this make is going to complete?". So I modified
linefan to save its input to a file and do an estimate of when the "make"
session would complete based on the saved previous output.

Typical usage would be like this:

  # Record a "successful" make session duration and number of lines of output:
  $ make 2>&1 | linefan -R linefan.log

  # When it ends the file linefan.log will contain the following entries:
  duration=<number of seconds it ran>
  target=<number of lines read from stdin>

  # Use the information from linefan.log to show the progress percentage and
  # estimated time remaining:
  $ make 2>&1 | linefan -R linefan.log
  -  89% 3m 2s

Another example:

  # Find out how long a typical "make" session takes
  $ begin=`date +%s`; make ; end=`date +%s`; ec

  # Show the progress percentage and the estimated remaining time
  $ make | linefan -d $((end - begin))

Needs age: https://github.com/holygeek/age.git
