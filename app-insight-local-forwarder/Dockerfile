FROM microsoft/dotnet:2.1-runtime

RUN mkdir /lf
ADD . /lf/
WORKDIR /lf/bin

EXPOSE 55678

ENTRYPOINT ["dotnet", "./Microsoft.LocalForwarder.ConsoleHost.dll", "noninteractive"]
