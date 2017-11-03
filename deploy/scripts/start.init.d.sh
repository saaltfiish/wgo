#! /bin/sh
#
# chkconfig: - 55 45
# description:  The wgo daemon start script
# processname: {wgo_app_name}

# Standard LSB functions
#. /lib/lsb/init-functions

# Source function library.
. /etc/init.d/functions

## env
#export TZ="Asia/Shanghai"

# Check that networking is up.
. /etc/sysconfig/network

if [ "$NETWORKING" = "no" ]
then
    exit 0
fi

RETVAL=0

# 工作目录, 一般为程序所在目录
workerdir="{path_to_daemon}"
# 运行文件
Daemon="${workerdir}/{exec_file_name}"
# 程序名, 一般为运行文件名, 也可以自定义
#prog=$(basename $Daemon)
prog="{custom_prog_name}"

pidfile="${workerdir}/run/${prog}.pid"
lockfile="${workerdir}/run/${prog}"

start () {
    echo -n $"Starting $prog: "

    daemon --pidfile ${pidfile} ${Daemon}
    RETVAL=$?
    echo
    [ $RETVAL -eq 0 ] && touch ${lockfile}
}
stop () {
    echo -n $"Stopping $prog: "
    killproc -p ${pidfile} ${prog}
    RETVAL=$?
    echo
    if [ $RETVAL -eq 0 ] ; then
        rm -f ${lockfile} ${pidfile}
    fi
}
reload () {
    echo -n $"Reloading $prog: "
    killproc -p ${pidfile} ${prog} -1
    RETVAL=$?
    echo
}

restart () {
        stop
        start
}


# See how we were called.
case "$1" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  reload)
    reload
    ;;
  status)
    status -p ${pidfile} ${prog}
    RETVAL=$?
    ;;
  restart)
    restart
    ;;
  condrestart|try-restart)
    [ -f ${lockfile} ] && restart || :
    ;;
  *)
    #echo $"Usage: $0 {start|stop|status|restart|reload|force-reload|condrestart|try-restart}"
    echo $"Usage: $0 {start|stop|status|restart|reload}"
    RETVAL=2
        ;;
esac

exit $RETVAL

