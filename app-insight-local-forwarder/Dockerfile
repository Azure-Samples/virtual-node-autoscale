FROM mcr.microsoft.com/dotnet/core/runtime:2.2.2-alpine3.8

RUN apk --no-cache add libc6-compat
RUN mkdir /lf
ADD . /lf/
WORKDIR /lf/bin

EXPOSE 55678

ENTRYPOINT ["dotnet", "./Microsoft.LocalForwarder.ConsoleHost.dll", "noninteractive"]
