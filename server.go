package main

import (
    "crypto/md5"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "time"
)

var logFile os.File
var listenAddr string
var outputDir string

func init() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    var logFilePath string
    var userOutputDir string
    flag.StringVar(&logFilePath, "log", "", "Path to log file. Default: no log file")
    flag.StringVar(&listenAddr, "addr", ":80", "Address on which the server listens. Default: :80")
    flag.StringVar(&userOutputDir, "out", "", "Path to output directory for uploaded files. Default: current working directory")

    flag.Parse()

    wd, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }

    // log to file if specified
    if len(logFilePath) > 0 {
        var path string
        if !filepath.IsAbs(logFilePath) {
            path = filepath.Join(wd, logFilePath)
        } else {
            path = logFilePath
        }

        logFile, err := os.OpenFile(path, os.O_APPEND | os.O_CREATE | os.O_WRONLY, 0666)
        if err != nil {
            log.Fatal(err)
        }

        log.SetOutput(logFile)
    }

    // set output directory
    if len(userOutputDir) > 0 {
        if !filepath.IsAbs(userOutputDir) {
            outputDir = filepath.Join(wd, userOutputDir)
        } else {
            outputDir = userOutputDir
        }
    } else {
        outputDir = wd
    }
}

func main() {
    defer logFile.Close()

    go httpServer()

    // make a channel to listen for interrupt/kill signals
    c := make(chan os.Signal, 1)
    defer close(c)
    signal.Notify(c, os.Interrupt, os.Kill)

    // block until signal received
    <-c
    log.Println("Stopping...")
}

func httpServer() {
    http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
        // parse the multipart form file upload; store up to 10 MB in memory
        r.ParseMultipartForm(10485760)

        // check that files exist
        if r.MultipartForm == nil || len(r.MultipartForm.File) < 1 {
            w.Write([]byte(htmlcode))
            return;
        }

        // loop through all FileHeaders
        for _, fhArray := range r.MultipartForm.File {
            for _, fh := range fhArray {
                // open file
                file, err := fh.Open()
                if err != nil {
                    log.Println(err)
                    break
                }
                defer file.Close()

                // compute md5 checksum of file
                h := md5.New()
                _, err = io.Copy(h, file)
                if err != nil {
                    log.Println(err)
                    break
                }

                // seek back to the beginning of the file
                if _, err = file.Seek(0, 0); err != nil {
                    log.Println(err)
                    break
                }

                // get the filename and path to write to
                outFileName := fmt.Sprintf("%s %x %s", time.Now().Format(`2006-01-02 15.04.05 -0700`), h.Sum(nil), fh.Filename)
                outFilePath := filepath.Join(outputDir, outFileName)
                outFile, err := os.OpenFile(outFilePath, os.O_CREATE | os.O_EXCL | os.O_WRONLY, 0666)
                if err != nil {
                    log.Println(err)
                    break
                }
                defer outFile.Close()

                // write the file
                _, err = io.Copy(outFile, file)
            }
        }
    })

    log.Printf("Starting http server on %s", listenAddr)
    if err := http.ListenAndServe(listenAddr, nil); err != nil {
        log.Fatal(err)
    }
}


// from YUI example: https://yuilibrary.com/yui/docs/uploader/uploader-dd.html
var htmlcode = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Simple Multiple Files Uploader with HTML5 Drag-and-Drop Support - YUI Library</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<script src="https://yui-s.yahooapis.com/3.18.1/build/yui/yui-min.js"></script>
</head>
<style>
#filelist {
    margin-top: 15px;
}

#uploadFilesButtonContainer, #selectFilesButtonContainer, #overallProgress {
    display: inline-block;
}

#overallProgress {
    float: right;
}

.yellowBackground {
  background: #F2E699;
}
</style>
<body>
<div id="exampleContainer">
<div id="uploaderContainer">
    <div id="selectFilesButtonContainer">
    </div>
    <div id="uploadFilesButtonContainer">
      <button type="button" id="uploadFilesButton"
              class="yui3-button" style="width:250px; height:35px;">Upload Files</button>
    </div>
    <div id="overallProgress">
    </div>
</div>

<div id="filelist">
  <table id="filenames">
    <thead>
       <tr><th>File name</th><th>File size</th><th>Percent uploaded</th></tr>
       <tr id="nofiles">
        <td colspan="3" id="ddmessage">
            <strong>No files selected.</strong>
        </td>
       </tr>
    </thead>
    <tbody>
    </tbody>
  </table>
</div>
</div>

<script>

YUI({filter:"raw"}).use("uploader", function(Y) {
Y.one("#overallProgress").set("text", "Uploader type: " + Y.Uploader.TYPE);
   if (Y.Uploader.TYPE != "none" && !Y.UA.ios) {
       var uploader = new Y.Uploader({width: "250px",
                                      height: "35px",
                                      multipleFiles: true,
                                      swfURL: "flashuploader.swf?t=" + Math.random(),
                                      uploadURL: "/",
                                      simLimit: 2,
                                      withCredentials: false
                                     });
       var uploadDone = false;

       if (Y.Uploader.TYPE == "html5") {
          uploader.set("dragAndDropArea", "body");

          Y.one("#ddmessage").setHTML("<strong>Drag and drop files here.</strong>");

          uploader.on(["dragenter", "dragover"], function (event) {
              var ddmessage = Y.one("#ddmessage");
              if (ddmessage) {
                ddmessage.setHTML("<strong>Files detected, drop them here!</strong>");
                ddmessage.addClass("yellowBackground");
              }
           });

           uploader.on(["dragleave", "drop"], function (event) {
              var ddmessage = Y.one("#ddmessage");
              if (ddmessage) {
                ddmessage.setHTML("<strong>Drag and drop files here.</strong>");
                ddmessage.removeClass("yellowBackground");
              }
           });
       }

       uploader.render("#selectFilesButtonContainer");

       uploader.after("fileselect", function (event) {

          var fileList = event.fileList;
          var fileTable = Y.one("#filenames tbody");
          if (fileList.length > 0 && Y.one("#nofiles")) {
            Y.one("#nofiles").remove();
          }

          if (uploadDone) {
            uploadDone = false;
            fileTable.setHTML("");
          }

          Y.each(fileList, function (fileInstance) {
              fileTable.append("<tr id='" + fileInstance.get("id") + "_row" + "'>" +
                                    "<td class='filename'>" + fileInstance.get("name") + "</td>" +
                                    "<td class='filesize'>" + fileInstance.get("size") + "</td>" +
                                    "<td class='percentdone'>Hasn't started yet</td>");
                             });
       });

       uploader.on("uploadprogress", function (event) {
            var fileRow = Y.one("#" + event.file.get("id") + "_row");
                fileRow.one(".percentdone").set("text", event.percentLoaded + "%");
       });

       uploader.on("uploadstart", function (event) {
            uploader.set("enabled", false);
            Y.one("#uploadFilesButton").addClass("yui3-button-disabled");
            Y.one("#uploadFilesButton").detach("click");
       });

       uploader.on("uploadcomplete", function (event) {
            var fileRow = Y.one("#" + event.file.get("id") + "_row");
                fileRow.one(".percentdone").set("text", "Finished!");
       });

       uploader.on("totaluploadprogress", function (event) {
                Y.one("#overallProgress").setHTML("Total uploaded: <strong>" +
                                                     event.percentLoaded + "%" +
                                                     "</strong>");
       });

       uploader.on("alluploadscomplete", function (event) {
                     uploader.set("enabled", true);
                     uploader.set("fileList", []);
                     Y.one("#uploadFilesButton").removeClass("yui3-button-disabled");
                     Y.one("#uploadFilesButton").on("click", function () {
                          if (!uploadDone && uploader.get("fileList").length > 0) {
                             uploader.uploadAll();
                          }
                     });
                     Y.one("#overallProgress").set("text", "Uploads complete!");
                     uploadDone = true;
       });

       Y.one("#uploadFilesButton").on("click", function () {
         if (!uploadDone && uploader.get("fileList").length > 0) {
            uploader.uploadAll();
         }
       });
   }
   else {
       Y.one("#uploaderContainer").set("text", "We are sorry, but to use the uploader, you either need a browser that support HTML5 or have the Flash player installed on your computer.");
   }


});

</script>
</body>
</html>`

