FROM ubuntu:20.04

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="faas.czlingo.buildpacks.stacks"
LABEL io.buildpacks.stack.id="faas.czlingo.buildpacks.stacks"