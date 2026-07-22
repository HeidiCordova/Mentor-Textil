#ifndef GSTREAMER_PIPELINE_H
#define GSTREAMER_PIPELINE_H

#include <jni.h>
#include <gst/gst.h>
#include <gst/video/video.h>
#include <android/native_window.h>
#include <android/native_window_jni.h>

typedef struct {
    GstElement *pipeline;
    GstElement *camerabin;
    GstElement *videoscale;
    GstElement *x264enc;
    GstElement *h264parse;
    GstElement *rtph264pay;
    GstElement *udpsink;
    GstElement *appsrc;  // Nuevo: para recibir datos H.264
    
    GstBus *bus;
    guint bus_watch_id;
    
    gint fps;
    gint64 bitrate;
    gint64 dropped_frames;
    gint64 latency;
    
    gboolean is_running;
} GStreamerPipeline;

// Funciones JNI
extern "C" {
    // Inicializar GStreamer (debe llamarse primero)
    jboolean Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeInit(
        JNIEnv *env, jclass clazz);
    
    jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeCreatePipeline(
        JNIEnv *env, jobject obj, jstring pipeline_str, jobject surface_view);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeStartPipeline(
        JNIEnv *env, jobject obj, jlong handle);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeStopPipeline(
        JNIEnv *env, jobject obj, jlong handle);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeDestroyPipeline(
        JNIEnv *env, jobject obj, jlong handle);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativePausePipeline(
        JNIEnv *env, jobject obj, jlong handle);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeResumePipeline(
        JNIEnv *env, jobject obj, jlong handle);
    
    jdouble Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetFps(
        JNIEnv *env, jobject obj, jlong handle);
    
    jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetBitrate(
        JNIEnv *env, jobject obj, jlong handle);
    
    jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetDroppedFrames(
        JNIEnv *env, jobject obj, jlong handle);
    
    jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetLatency(
        JNIEnv *env, jobject obj, jlong handle);
    
    void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativePushH264Data(
        JNIEnv *env, jobject obj, jlong handle, jbyteArray data, jint size, jlong timestamp);
}

#endif // GSTREAMER_PIPELINE_H
