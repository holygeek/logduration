#!/bin/sh
#vim:ts=8:
set -e
me=$(basename $0)
args="$(basename $0) $@"

usage() {
	local default_format_x
	default_format_x=$(echo "$format_x"|paste -s -d'X'|sed -e 's/X/\\\\n/g')
	echo "Usage: $me [-h] [-t <black|white>] [-W width] [-H height] <log.txt>

log.txt is the file produced by logduration

OPTIONS
    -b
	Batch mode - do not open the generated png file.

    -c
	Log the cumulative value

    -d <duration>
	Limit xrange max to <duration>, e.g: -d '1 hour'

    -E <end>
	Set xrange max to <end>. Use with -S.

    -f <time format>
	Value to use as timefmt. Default is '%d/%m/%y %H:%M:%S'

    -h
	Show this help message

    -n
	Dry run - show what is going to be executed.

    -o <file.png>
	Use <file.png> as the graph filename instead of <log.txt>.png

    -S <start>
	Set xrange min to <start>. Use with -E.

    -s <since>
	Limit xrange min to <since>, e.g: -s '1 day ago'

    -T <title prefix>

    -t <black|white>
	Use black or white theme. Default is white.

    -W <width>

    -H <height>

    -x <format_x>
        Format for x ticks, default is $default_format_x"
}

titlePrefix=""
width=1000
height=500
theme=white
since=
format_x='%l%p\n%a\n%d'
duration=
dryrun=
UNTIL=
SINCE=
timefmt='%Y/%m/%d %H:%M:%S'
png_file=
batchmode=
while getopts bd:E:f:hno:W:H:S:s:T:t:x: opt
do
	case "$opt" in
		b)
			batchmode=t
			;;
		d)
			duration=$OPTARG
			;;
		E)
			UNTIL=$OPTARG
			;;
		f)
			timefmt=$OPTARG
			;;
		n)
			dryrun=t
			;;
		W)
			width=$OPTARG
			;;
		H)
			height=$OPTARG
			;;
		h)
			usage
			exit
			;;
		o)
			png_file=$OPTARG
			;;
		S)
			SINCE=$OPTARG
			;;
		s)
			since=$OPTARG
			;;
		T)
			titlePrefix="$OPTARG "
			;;
		t)
			theme="$OPTARG"
			;;
		x)
			format_x=$OPTARG
			;;
		\?)
			echo "$me: Unknown option '$opt'"
			exit 1
			;;
	esac
done
shift $(($OPTIND -1))

if [ -z "$1" ]; then
  usage
  exit 1
fi

if [ "$theme" = "black" ]; then
  fg=white
  bg=black
elif [ "$theme" = "white" ]; then
  bg=white
  fg=black
else
  echo "Unknown theme: $theme"
  exit 1
fi


log=$1

# Figure out time format
first_time=$(head -2 "$log"|tail -1|awk '{print $2}')
case "$first_time" in
	*:*:*)
		timefmt='%Y/%m/%d %H:%M:%S'
esac

#png_file=$log-$(date -Iseconds).png
if [ -z "$png_file" ]; then
	png_file=$log.png
fi
xmin=
xmax=

#title="${titlePrefix}Log Frequency from $(head -2 "$log"|tail -1|awk '{print $1" "$2}') - $log"
if [ -n "$SINCE" ]; then
	xmin="\"$SINCE\""
	#title="${titlePrefix}Log Frequency from $SINCE - $log"
	if [ -n "$UNTIL" ]; then
		xmax="\"$UNTIL\""
		#title="${titlePrefix}Log Frequency from $SINCE - $UNTIL - $log"
	fi
	log_start=$SINCE
elif [ -n "$since" ]; then
	xmin=\"$(date --utc -d "$since" +"$timefmt")\"
	if [ -n "$duration" ]; then
		xmax=\"$(date --utc -d "$since + $duration" +"$timefmt")\"
	fi
fi

#title="$args"
title=$(basename $0)
for i in "$@"; do
  title="$title \"$i\""
done

if [ -n "$xmin" -o -n "$xmax" ]; then
	xrange='set xrange ['$xmin':'$xmax']'
fi

using="using 1:\"duration(ms)\" with impulses"

gnuplot_cmd="set xdata time;
set timefmt \"$timefmt\";
set title '$title' textcolor rgb \"$fg\";
set key autotitle columnhead textcolor rgb \"$fg\";
set format x \"$format_x\";
$xrange;
set xlabel \"Time\" textcolor rgb \"$fg\";
set ylabel \"Duration (ms)\" textcolor rgb \"$fg\";
set yrange [0:];
set terminal png size $width,$height background rgb\"$bg\";
set border lw 1 lc rgb \"$fg\";
set xtics textcolor rgb \"$fg\";
set ytics textcolor rgb \"$fg\";
set grid linecolor rgb \"gray\";

plot '$log' $using;"

if [ -n "$dryrun" ]; then
  echo "gnuplot -persist -e \"$gnuplot_cmd\""
else
  gnuplot -persist -e "$gnuplot_cmd" >$png_file
  if [ -z "$batchmode" ]; then
    echo qiv "$png_file" &&
    qiv "$png_file"
  fi
fi
