package api

import (
	"encoding/json"
	"fmt"
)

type Stats struct {
	Cameras      map[string]CameraStats `json:"-"`
	Detectors    map[string]interface{} `json:"detectors"`
	DetectionFps float64                `json:"detection_fps"`
	GpuUsages    map[string]interface{} `json:"gpu_usages"`
	CpuUsages    map[string]interface{} `json:"cpu_usages"`
	Service      map[string]interface{} `json:"service"`
}

func (stats Stats) String() string {
	return fmt.Sprintf("cameras: %s", stats.Cameras)
}

type CameraStats struct {
	CameraFps        float64 `json:"camera_fps"`
	ProcessFps       float64 `json:"process_fps"`
	SkippedFps       float64 `json:"skipped_fps"`
	DetectionFps     float64 `json:"detection_fps"`
	DetectionEnabled int     `json:"detection_enabled"`
	Pid              int     `json:"pid"`
	CapturePid       int     `json:"capture_pid"`
	FfmpegPid        int     `json:"ffmpeg_pid"`
	Service          struct {
		Uptime        int64                  `json:"uptime"`
		Version       string                 `json:"version"`
		LatestVersion string                 `json:"latest_version"`
		Storage       map[string]interface{} `json:"storage"`
		Temperatures  map[string]interface{} `json:"temperatures"`
		LastUpdated   int64                  `json:"last_updated"`
	} `json:"service"`
}

func (camera CameraStats) String() string {
	return fmt.Sprintf("fps: %f, proc-fps: %f, skip-fps: %f, detection-fps: %f, detection-enabled: %t", camera.CameraFps, camera.ProcessFps, camera.SkippedFps, camera.DetectionFps, camera.DetectionEnabled == 1)
}

func (stats *Stats) UnmarshalJSON(data []byte) error {
	type Tmp Stats
	var tmp Tmp
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*stats = Stats(tmp)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	delete(raw, "detectors")
	delete(raw, "detection_fps")
	delete(raw, "gpu_usages")
	delete(raw, "cpu_usages")
	delete(raw, "service")

	stats.Cameras = make(map[string]CameraStats)
	for name, raw := range raw {
		var camera CameraStats
		if err := json.Unmarshal(raw, &camera); err != nil {
			return err
		}
		stats.Cameras[name] = camera
	}

	return nil
}

// {
//     "camera1": {
//         "camera_fps": 5.1,
//         "process_fps": 5.1,
//         "skipped_fps": 0.0,
//         "detection_fps": 0.0,
//         "detection_enabled": 1,
//         "pid": 294,
//         "capture_pid": 299,
//         "ffmpeg_pid": 337
//     },
//     "camera2": {
//         "camera_fps": 5.1,
//         "process_fps": 5.1,
//         "skipped_fps": 0.0,
//         "detection_fps": 0.0,
//         "detection_enabled": 1,
//         "pid": 296,
//         "capture_pid": 302,
//         "ffmpeg_pid": 339
//     },
//     "detectors": {
//         "coral": {
//             "inference_speed": 9.65,
//             "detection_start": 0.0,
//             "pid": 3445175
//         }
//     },
//     "detection_fps": 0.0,
//     "gpu_usages": {
//         "intel-vaapi": {
//             "gpu": "10.71 %",
//             "mem": "- %"
//         }
//     },
//     "cpu_usages": {
//         "top": {
//             "cpu": "users,",
//             "mem": "load"
//         },
//         "Tasks:": {
//             "cpu": "stopped,",
//             "mem": "0"
//         },
//         "%Cpu(s):": {
//             "cpu": "id,",
//             "mem": "1.3"
//         },
//         "MiB": {
//             "cpu": "13200.4",
//             "mem": "avail"
//         },
//         "PID": {
//             "cpu": "%CPU",
//             "mem": "%MEM"
//         },
//         "337": {
//             "cpu": "13.7",
//             "mem": "0.7"
//         },
//         "339": {
//             "cpu": "12.0",
//             "mem": "0.7"
//         },
//         "299": {
//             "cpu": "6.0",
//             "mem": "0.6"
//         },
//         "291": {
//             "cpu": "1.0",
//             "mem": "0.6"
//         },
//         "294": {
//             "cpu": "0.7",
//             "mem": "0.9"
//         },
//         "302": {
//             "cpu": "4.7",
//             "mem": "0.7"
//         },
//         "1": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "15": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "16": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "24": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "25": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "26": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "27": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "28": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "29": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "30": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "31": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "32": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "41": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "42": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "80": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "81": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "82": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "88": {
//             "cpu": "0.0",
//             "mem": "0.2"
//         },
//         "102": {
//             "cpu": "2.7",
//             "mem": "3.1"
//         },
//         "110": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "118": {
//             "cpu": "0.0",
//             "mem": "0.1"
//         },
//         "134": {
//             "cpu": "0.0",
//             "mem": "0.2"
//         },
//         "135": {
//             "cpu": "0.0",
//             "mem": "0.3"
//         },
//         "136": {
//             "cpu": "0.0",
//             "mem": "0.3"
//         },
//         "142": {
//             "cpu": "0.0",
//             "mem": "0.3"
//         },
//         "282": {
//             "cpu": "0.0",
//             "mem": "0.4"
//         },
//         "288": {
//             "cpu": "0.0",
//             "mem": "0.1"
//         },
//         "295": {
//             "cpu": "0.0",
//             "mem": "0.4"
//         },
//         "296": {
//             "cpu": "0.7",
//             "mem": "1.0"
//         },
//         "301": {
//             "cpu": "0.0",
//             "mem": "0.4"
//         },
//         "306": {
//             "cpu": "0.0",
//             "mem": "0.1"
//         },
//         "309313": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "309324": {
//             "cpu": "0.0",
//             "mem": "0.0"
//         },
//         "3445175": {
//             "cpu": "0.0",
//             "mem": "3.0"
//         }
//     },
//     "service": {
//         "uptime": 3192117,
//         "version": "0.12.1-367d724",
//         "latest_version": "unknown",
//         "storage": {
//             "/media/frigate/recordings": {
//                 "total": 501386.0,
//                 "used": 463132.5,
//                 "free": 12709.3,
//                 "mount_type": "ext4"
//             },
//             "/media/frigate/clips": {
//                 "total": 501386.0,
//                 "used": 463132.5,
//                 "free": 12709.3,
//                 "mount_type": "ext4"
//             },
//             "/tmp/cache": {
//                 "total": 1000.0,
//                 "used": 7.1,
//                 "free": 992.9,
//                 "mount_type": "tmpfs"
//             },
//             "/dev/shm": {
//                 "total": 209.7,
//                 "used": 33.5,
//                 "free": 176.2,
//                 "mount_type": "tmpfs"
//             }
//         },
//         "temperatures": {},
//         "last_updated": 1720473390
//     }
// }
