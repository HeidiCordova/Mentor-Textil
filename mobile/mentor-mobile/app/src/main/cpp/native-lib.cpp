#include <jni.h>
#include <string>

extern "C" JNIEXPORT jstring JNICALL
Java_com_mentor_mobile_MainActivity_stringFromJNI(
        JNIEnv* env,
        jobject /* this */) {
    std::string hello = "GStreamer listo en MentorMobile";
    return env->NewStringUTF(hello.c_str());
}