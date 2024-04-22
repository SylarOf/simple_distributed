package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func unzipFile(zipFile, destDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), os.ModePerm)
			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// 解析表单
	err := r.ParseMultipartForm(10 << 20) // 设置上传文件大小的最大限制为 10MB
	if err != nil {
		http.Error(w, "解析表单失败", http.StatusInternalServerError)
		return
	}

	// 获取上传的文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "获取上传文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建一个目录来保存上传的文件
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, os.ModePerm)
	}

	// 创建一个新的文件来保存上传的 ZIP 文件
	filePath := filepath.Join(uploadDir, handler.Filename)
	f, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "创建文件失败", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// 将上传的文件保存到本地
	_, err = io.Copy(f, file)
	if err != nil {
		http.Error(w, "保存文件失败", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dd", http.StatusSeeOther)
	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("文件上传成功"))

}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// 设置下载文件的名称
	fileName := "download.zip"

	// 设置响应头，告诉浏览器要下载文件
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/zip")

	// 创建一个新的 ZIP 文件
	zipFile := zip.NewWriter(w)
	defer zipFile.Close()

	// 添加上传的文件到 ZIP 文件中
	uploadDir := "./uploads"
	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// 获取文件的相对路径
			relPath, err := filepath.Rel(uploadDir, path)
			if err != nil {
				return err
			}

			// 创建一个新的 ZIP 文件条目
			zipEntry, err := zipFile.Create(relPath)
			if err != nil {
				return err
			}

			// 将文件内容写入 ZIP 文件条目
			_, err = io.Copy(zipEntry, file)
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, "创建 ZIP 文件失败", http.StatusInternalServerError)
		return
	}
}
func answerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello world\n")

	// 指定上传的目录和目标解压缩目录
	uploadDir := "./uploads"
	destDir := "./lockbud/Code/uploads"

	// 获取上传目录下的所有文件
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "读取上传目录失败", http.StatusInternalServerError)
		return
	}

	// 遍历上传目录下的文件，找到第一个 .zip 文件
	var zipFile string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".zip") {
			zipFile = filepath.Join(uploadDir, file.Name())
			break
		}
	}

	// 检查是否找到了 .zip 文件
	if zipFile == "" {
		http.Error(w, "未找到上传的 .zip 文件", http.StatusInternalServerError)
		return
	}

	// 解压缩 .zip 文件到目标目录
	err = unzipFile(zipFile, destDir)
	if err != nil {
		http.Error(w, "解压缩文件失败", http.StatusInternalServerError)
		return
	}

	// 输出成功信息到页面
	fmt.Fprintf(w, "成功解压缩上传的 .zip 文件到 %s 目录", destDir)

	{
		// 执行 ls 命令
        cmd := exec.Command("sh", "-c", "cd lockbud/Code; ./detect.sh uploads/use_after_free")
		// 获取命令的输出结果
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintf(w, "命令执行失败:", err)
			return
		}

		// 将输出结果转换为字符串并打印出来
		fmt.Fprintf(w, string(output))
	}
}
func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))
	// 上传文件的路由
	http.HandleFunc("/upload", uploadHandler)

	// 下载文件的路由
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/answer", answerHandler)

	// 启动服务器
	http.ListenAndServe(":8080", nil)
}
