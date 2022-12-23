# 本文件仅限在linux上运行
# 我们在github action里用到了本文件，用于 自动编译发布包。

# 我们这里规定，使用该 Makefile_release.mk 进行make时，必须要明确指明 BUILD_VERSION。因为是发布包嘛。

#BUILD_VERSION   := vx.x.x-beta.x 这个将在github action里自动通过tag配置, 参见 .github/workflows/build_release.yml

prefix :=verysimple

cmd:=go build -tags "$(tags)"  -trimpath -ldflags "-X 'main.Version=${BUILD_VERSION}' -s -w -buildid="  -o


ifdef PACK
define compile
	CGO_ENABLED=0 GOOS=$(2) GOARCH=$(3) GOAMD64=$(4) $(cmd) ${prefix}_$(1)
	mv ${prefix}_$(1) verysimple$(5)
	tar -cJf ${prefix}_$(1).tar.xz verysimple$(5) -C ../../ examples/
	rm verysimple$(5)
endef

else

define compile
	CGO_ENABLED=0 GOOS=$(2) GOARCH=$(3) GOAMD64=$(4) $(cmd) ${prefix}_$(1)$(5)
endef
endif


main: linux_amd64_v1 linux_amd64_v2 linux_amd64_v3 windows_amd64_v1 windows_amd64_v2 windows_amd64_v3

# 注意调用参数时，逗号前后不能留空格
# 关于arm版本号 https://github.com/goreleaser/goreleaser/issues/36

linux_amd64_v1:
	$(call compile,linux_amd64_v1,linux,amd64,v1)
linux_amd64_v2:
	$(call compile,linux_amd64_v2,linux,amd64,v2)
linux_amd64_v3:
	$(call compile,linux_amd64_v3,linux,amd64,v3)
windows_amd64_v1:
	$(call compile,windows_amd64_v1,windows,amd64,v1,.exe)
windows_amd64_v2:
	$(call compile,windows_amd64_v2,windows,amd64,v2,.exe)
windows_amd64_v3:
	$(call compile,windows_amd64_v3,windows,amd64,v3,.exe)


clean:
	rm -f ${prefix}
	rm -f ${prefix}.exe
	rm -f ${prefix}_*
	rm -f *.tar.xz
