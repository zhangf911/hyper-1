package daemon

import (
	"fmt"

	"hyper/engine"
	"hyper/lib/glog"
	"hyper/types"
)

func (daemon *Daemon) CmdPodRm(job *engine.Job) (err error) {
    var (
        podId = job.Args[0]
        pod   = daemon.podList[podId]
        code  = 0
        cause = ""
    )
    if pod == nil {
        return fmt.Errorf("Can not find that Pod(%s)", podId)
    }

	if daemon.podList[podId].Status != types.S_POD_RUNNING {
        // If the pod type is kubernetes, we just remove the pod from the pod list.
        // The persistent data has been removed since we got the E_VM_SHUTDOWN event.
        if daemon.podList[podId].Type == "kubernetes" {
            daemon.RemovePod(podId)
            code = types.E_OK
        } else {
            daemon.DeletePodFromDB(podId)
            for _, c := range pod.Containers {
                glog.V(1).Infof("Ready to rm container: %s", c.Id)
                if _, _, err = daemon.dockerCli.SendCmdDelete(c.Id); err != nil {
                    glog.V(1).Infof("Error to rm container: %s", err.Error())
                }
            }
            daemon.RemovePod(podId)
            daemon.DeletePodContainerFromDB(podId)
            daemon.DeleteVolumeId(podId)
            code = types.E_OK
        }
	} else {
        code, cause, err = daemon.StopPod(podId, "yes")
        if err != nil {
            return err
        }
        if code == types.E_VM_SHUTDOWN {
            daemon.DeletePodFromDB(podId)
            for _, c := range pod.Containers {
                glog.V(1).Infof("Ready to rm container: %s", c.Id)
                if _, _, err = daemon.dockerCli.SendCmdDelete(c.Id); err != nil {
                    glog.V(1).Infof("Error to rm container: %s", err.Error())
                }
            }
            daemon.RemovePod(podId)
            daemon.DeletePodContainerFromDB(podId)
            daemon.DeleteVolumeId(podId)
        }
        code = types.E_OK
    }

	// Prepare the qemu status to client
	v := &engine.Env{}
	v.Set("ID", podId)
	v.SetInt("Code", code)
	v.Set("Cause", cause)
	if _, err = v.WriteTo(job.Stdout); err != nil {
		return err
	}

	return nil
}
