# syntax=docker/dockerfile:1

# Build the application on the builder's native architecture and cross-compile
# it for the requested image architecture.
FROM --platform=$BUILDPLATFORM golang:1.25.4 AS dmp-builder

ARG GOPROXY
ARG GOSUMDB
ARG GOPRIVATE
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build -ldflags '-s -w' -v -o dmp main.go

# Build emulators only for ARM64 targets. This stage runs on TARGETPLATFORM,
# which lets Buildx use build-time emulation when the builder is AMD64.
FROM ubuntu:24.04 AS compat-builder

ARG TARGETARCH
ARG BOX64_VERSION=v0.4.4
ARG BOX64_COMMIT=2f130fab1d6e1a4ee8a71dc60cfdfcc839ad192a
ARG BOX86_VERSION=v0.3.8
ARG BOX86_COMMIT=ec278bd7843252d7bfc6e5e91d7074f9c0310de7

RUN set -eux; \
    mkdir -p \
        /opt/box-install/usr/local \
        /opt/box-install/usr/lib/box64-x86_64-linux-gnu \
        /opt/box-install/usr/lib/box64-i386-linux-gnu; \
    if [ "${TARGETARCH}" != "arm64" ]; then exit 0; fi; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        cmake \
        gcc-arm-linux-gnueabihf \
        git \
        python3 \
        libc6-dev-armhf-cross; \
    git clone --depth 1 --branch "${BOX64_VERSION}" https://github.com/ptitSeb/box64.git /tmp/box64; \
    test "$(git -C /tmp/box64 rev-parse HEAD)" = "${BOX64_COMMIT}"; \
    cmake -S /tmp/box64 -B /tmp/box64/build \
        -DRK3588=1 \
        -DBOX32=ON \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/usr/local; \
    cmake --build /tmp/box64/build --parallel 4; \
    DESTDIR=/opt/box-install cmake --install /tmp/box64/build; \
    git clone --depth 1 --branch "${BOX86_VERSION}" https://github.com/ptitSeb/box86.git /tmp/box86; \
    test "$(git -C /tmp/box86 rev-parse HEAD)" = "${BOX86_COMMIT}"; \
    cmake -S /tmp/box86 -B /tmp/box86/build \
        -DRK3588=1 \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_SYSTEM_NAME=Linux \
        -DCMAKE_SYSTEM_PROCESSOR=armv7l \
        -DCMAKE_C_COMPILER=arm-linux-gnueabihf-gcc \
        -DCMAKE_INSTALL_PREFIX=/usr/local; \
    cmake --build /tmp/box86/build --parallel 4; \
    DESTDIR=/opt/box-install cmake --install /tmp/box86/build; \
    rm -rf /var/lib/apt/lists/* /tmp/box64 /tmp/box86

# Build the native SDL3 runtime payload only for ARM64, where Box64's SDL3
# wrapper needs a host-architecture library. Other target architectures keep
# an empty payload so the runtime COPY remains unconditional.
FROM ubuntu:24.04 AS sdl3-builder

ARG TARGETARCH
ARG SDL3_TAG=release-3.4.18
ARG SDL3_COMMIT=829a65d769d935c4852f8159e964312c0957260a

RUN set -eux; \
    mkdir -p /opt/sdl3-runtime/usr/local/lib; \
    if [ "${TARGETARCH}" != "arm64" ]; then exit 0; fi; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        cmake \
        git \
        libx11-dev \
        libxext-dev \
        libxcursor-dev \
        libxi-dev \
        libxrandr-dev \
        libxss-dev \
        libxtst-dev; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    if [ "${TARGETARCH}" != "arm64" ]; then exit 0; fi; \
    git clone --depth 1 --branch "${SDL3_TAG}" https://github.com/libsdl-org/SDL.git /tmp/SDL; \
    test "$(git -C /tmp/SDL rev-parse HEAD)" = "${SDL3_COMMIT}"; \
    cmake -S /tmp/SDL -B /tmp/SDL/build \
        -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_INSTALL_PREFIX=/usr/local \
        -DCMAKE_INSTALL_LIBDIR=lib \
        -DSDL_SHARED=ON \
        -DSDL_STATIC=OFF \
        -DSDL_TEST_LIBRARY=OFF \
        -DSDL_TESTS=OFF \
        -DSDL_EXAMPLES=OFF \
        -DSDL_INSTALL_TESTS=OFF \
        -DSDL_DISABLE_INSTALL_DOCS=ON; \
    cmake --build /tmp/SDL/build --parallel 4; \
    DESTDIR=/opt/sdl3-install cmake --install /tmp/SDL/build; \
    test -e /opt/sdl3-install/usr/local/lib/libSDL3.so.0; \
    cp -a /opt/sdl3-install/usr/local/lib/libSDL3.so* /opt/sdl3-runtime/usr/local/lib/; \
    test -e /opt/sdl3-runtime/usr/local/lib/libSDL3.so.0; \
    rm -rf /tmp/SDL /opt/sdl3-install

# Runtime base and user-facing container contract remain the existing Ubuntu
# image, /root work directory, entrypoint, port, and DMP defaults.
FROM ubuntu:24.04 AS runtime

ARG TARGETARCH

WORKDIR /root

RUN set -eux; \
    if [ "${TARGETARCH}" = "arm64" ]; then \
        dpkg --add-architecture armhf; \
    else \
        dpkg --add-architecture i386; \
    fi; \
    apt-get update; \
    if [ "${TARGETARCH}" = "arm64" ]; then \
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
            libcurl3t64-gnutls \
            libcurl3t64-gnutls:armhf \
            libc6:armhf \
            libstdc++6 \
            libsdl2-2.0-0 \
            file \
            libgcc-s1:armhf \
            libstdc++6:armhf \
            screen \
            wget \
            ca-certificates; \
    else \
        DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
            libcurl3t64-gnutls \
            lib32gcc-s1 \
            lib32stdc++6 \
            libcurl4-gnutls-dev:i386 \
            file \
            screen \
            wget \
            ca-certificates; \
    fi; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*

COPY --from=dmp-builder /app/dmp /root/dmp
COPY --from=compat-builder /opt/box-install/usr/local/ /usr/local/
COPY --from=compat-builder /opt/box-install/usr/lib/box64-x86_64-linux-gnu/ /usr/lib/box64-x86_64-linux-gnu/
COPY --from=compat-builder /opt/box-install/usr/lib/box64-i386-linux-gnu/ /usr/lib/box64-i386-linux-gnu/
COPY --from=sdl3-builder /opt/sdl3-runtime/usr/local/ /usr/local/
COPY docker/entry-point.sh /root/entry-point.sh

# Keep Valve's official SteamCMD archive and its linux32/steamcmd layout.
RUN mkdir -p /root/steamcmd /root/data /root/.klei && \
    wget -q https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz \
        -O /tmp/steamcmd_linux.tar.gz && \
    tar -xzf /tmp/steamcmd_linux.tar.gz -C /root/steamcmd && \
    rm -f /tmp/steamcmd_linux.tar.gz && \
    chmod +x /root/dmp /root/entry-point.sh && \
    test -x /root/steamcmd/steamcmd.sh && \
    test -x /root/steamcmd/linux32/steamcmd

# Fail the image build if Go emitted the wrong architecture or the expected
# ARM64 compatibility executables are missing from the ARM64 image.
RUN set -eux; \
    if [ "${TARGETARCH}" = "arm64" ]; then \
        file /root/dmp | grep -q "ARM aarch64"; \
        test -x /usr/local/bin/box64; \
        test -x /usr/local/bin/box86; \
        test -f /usr/lib/box64-x86_64-linux-gnu/libstdc++.so.6; \
        test -f /usr/lib/box64-i386-linux-gnu/libstdc++.so.6; \
        file -L /usr/lib/box64-x86_64-linux-gnu/libstdc++.so.6 | grep -q "x86-64"; \
        file -L /usr/lib/box64-i386-linux-gnu/libstdc++.so.6 | grep -qi "Intel 80386"; \
        test -e /usr/local/lib/libSDL3.so.0; \
        file -L /usr/local/lib/libSDL3.so.0 | grep -q "ARM aarch64"; \
        ldconfig; \
        ldconfig -p | grep -q "libSDL3.so.0"; \
        if ldd /usr/local/lib/libSDL3.so.0 | grep -q "not found"; then ldd /usr/local/lib/libSDL3.so.0; exit 1; fi; \
    else \
        file /root/dmp | grep -q "x86-64"; \
        test ! -e /usr/local/bin/box64; \
        test ! -e /usr/local/bin/box86; \
        test ! -e /usr/local/lib/libSDL3.so.0; \
    fi; \
    file /root/steamcmd/linux32/steamcmd | grep -qi "Intel 80386"

# 环境变量
# - 平台暴露端口，默认为80
ENV DMP_PORT=80
# 是否以容器方式启动
ENV DMP_IN_CONTAINER=1

# web端口
EXPOSE 80/tcp

# 设置容器启动时执行的脚本
ENTRYPOINT ["/root/entry-point.sh"]
