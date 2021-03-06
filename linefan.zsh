#!/usr/bin/zsh

usage() {
    echo -n "DESCRIPTION
  linefan - Spinning fan progress meter.
SYNOPSIS
  linefan [-h] [-c] [-d N] [-e N] [-R <file>] [-PO <file>] [-t N]
OPTIONS
  -c
    Clear the output when done. Leave no trace that it was run.

  -d N
    Show estimated time remaining based on N seconds total runtime.
    Setting N to 0 turns off the remaining time estimation. If runtime
    exceeds N, ?s is shown.

  -e N
    Update every N lines. Default is 1.

  -R <file>
    If files does not exist, record the duration and number of lines read from
    stdin into <file>. If file exist, use the values in <file> for the -t and
    -d arguments.

  -O <file>
    Write fan output to <file> instead of stdout. Useful for use with -P to
    show the lines read and progress in another tty:

        foo | linefan -PO /dev/pts/20

  -P
    Echo whatever was read from stdin to stdout.

  -t N
    Show estimated completion percentage based on N lines of max input.
    Setting N to 0 turns off the percentage estimation. If input lines
    is more than N, ?% will be shown instead.
"
}

clear=
every=1
target=0
record=
outfile=
duration=0
passthrough=
while getopts hcd:e:O:PR:t: opt
do
  case "$opt" in
    e) every="$OPTARG";;
    d) duration="$OPTARG";;
    t) target="$OPTARG";; # target nlines
    c) clear=yes;;
    O) outfile="$OPTARG";;
    P) passthrough=t;;
    R) record="$OPTARG";;
    h) usage ; exit;;
    \?) usage; exit;;
  esac
done
shift $(($OPTIND -1))

if [ -f "$record" ]; then
  saved_duration=0
  saved_target=0
  eval `cat $record`
  test ${saved_duration:-0} -gt 0 && duration=$saved_duration
  test ${saved_target:-0} -gt 0 && target=$saved_target
fi

if [ $duration -gt 0 -o $target -gt 0 -o -n "$record" ]; then
  start_time=`date +%s`
fi

echoerr() {
  echo $* >&2
}

percent=
output=
eta=
n=0
nlines=0 # Lines read so far
while read line; do
  if [ -n "$passthrough" ]; then
    echo $line
  fi
  nlines=$((nlines + 1))
  if [ $((nlines % $every)) -ne 0 ]; then
    continue
  fi

  r=$((n % 8))
  case $r in
          0|4) fan=-  ;;
          1|5) fan=\\ ;;
          2|6) fan=\| ;;
          3|7) fan=/  ;;
  esac
  n=$((n + 1))

  if [ $target -gt 0 ]; then
    percent=$((${nlines} * 100 / target))
    if (( $percent > 100 )); then
      #       '100%'
      percent='  ?%'
    else
      percent=" $percent%"
    fi

    if [ $duration -eq 0 -a $nlines -gt 0 ]; then
      lapsed=$((`date +%s` - start_time))
      # estimate the remaining time
      if [ $lapsed -gt 0 ]; then
        velocity=$(($nlines.0 / lapsed))
        remaining_distance=$((target - nlines))
        # Add 1 second to offset the last 0 second
        eta=`echo $(($remaining_distance / $velocity + 1))|sed -e 's/\..*//' -e 's/^-//'`
        eta="(`age -c -d $eta`)"
      fi
    elif [ $duration -gt 0 -o -n "$record" ]; then
      # Calculate remaining time based on given duration
      lapsed=$((`date +%s` - start_time))
      if [ $lapsed -gt $duration ]; then
        #eta='20m20s'
          eta='?s'
      else
          eta="`age -c -d $((duration - lapsed))`"
      fi
    fi
  fi

  output=`printf "%s %-5s %-8s    " $fan $percent $eta`
  if [ -n "$outfile" ]; then
    echoerr -ne "$output\e[${#output}D" >> $outfile
  else
    echoerr -ne "$output\e[${#output}D"
  fi
done
echoerr -ne "\e[${#output}C"

if [ -n "$record" -a ! -f $record ]; then
  dir=`dirname $record`
  if [ ! -d $dir ]; then
    mkdir -p $dir || echo -e "duration=$lapsed\ntarget=$nlines"
  fi
  echo -e "saved_duration=$lapsed\nsaved_target=$nlines" > $record
fi

if [ -n "$clear" ]; then
  # Leave no trace that we've ran
  spaces="                  "
  echoerr -ne "\e[${#output}D$spaces\e[${#spaces}D"
else
  echoerr
fi
