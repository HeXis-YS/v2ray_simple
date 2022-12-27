# 本文件仅限在linux上运行
# 我们在github action里用到了本文件，用于 自动编译发布包。

# 我们这里规定，使用该 Makefile_release.mk 进行make时，必须要明确指明 BUILD_VERSION。因为是发布包嘛。

#BUILD_VERSION   := vx.x.x-beta.x 这个将在github action里自动通过tag配置, 参见 .github/workflows/build_release.yml

ifdef LITE
prefix :=verysimple_lite
tags := notun,noquic,nocli
else
prefix :=verysimple
endif

cmd:=go build -tags "$(tags)"  -trimpath -ldflags "-X 'main.Version=${BUILD_VERSION}' -s -w -buildid="  -o


define compile
	CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) GOAMD64=$(3) $(cmd) ${prefix}_$(1)_$(2)_$(3)$(4)
endef


main: linux_amd64_v2 linux_amd64_v3 windows_amd64_v2 windows_amd64_v3

# 注意调用参数时，逗号前后不能留空格
# 关于arm版本号 https://github.com/goreleaser/goreleaser/issues/36

linux_amd64_v2:
	$(call compile,linux,amd64,v2)
linux_amd64_v3:
	$(call compile,linux,amd64,v3)
windows_amd64_v2:
	$(call compile,windows,amd64,v2,.exe)
windows_amd64_v3:
	$(call compile,windows,amd64,v3,.exe)


clean:
	rm -f ${prefix}
	rm -f ${prefix}.exe
	rm -f ${prefix}_*
