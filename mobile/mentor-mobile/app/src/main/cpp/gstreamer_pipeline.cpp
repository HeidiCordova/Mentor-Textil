#include "gstreamer_pipeline.h"
#include <android/log.h>
#include <cstring>
#include <ctime>
#include <gst/app/gstappsrc.h>

#define LOG_TAG "GStreamerPipeline"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

// Variable global para verificar si GStreamer está inicializado
static gboolean gstreamer_initialized = FALSE;

// Declarar funciones de registro de plugins
// Estas funciones están en las bibliotecas .a de los plugins
extern "C" {
    gboolean gst_plugin_coreelements_register(GstPlugin *plugin);
    gboolean gst_plugin_videoconvertscale_register(GstPlugin *plugin);
    gboolean gst_plugin_x264_register(GstPlugin *plugin);
    gboolean gst_plugin_videoparsersbad_register(GstPlugin *plugin);
    gboolean gst_plugin_rtp_register(GstPlugin *plugin);
    gboolean gst_plugin_udp_register(GstPlugin *plugin);
    // gboolean gst_plugin_androidmedia_register(GstPlugin *plugin); // Comentado: requiere OpenGL
    gboolean gst_plugin_app_register(GstPlugin *plugin);
    gboolean gst_plugin_videotestsrc_register(GstPlugin *plugin);
}

// Inicializar GStreamer (debe llamarse una vez al inicio)
jboolean Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeInit(
    JNIEnv *env, jclass clazz) {
    
    if (gstreamer_initialized) {
        LOGI("GStreamer already initialized");
        return JNI_TRUE;
    }
    
    LOGI("Initializing GStreamer...");
    
    // Inicializar GStreamer
    GError *error = NULL;
    if (!gst_init_check(NULL, NULL, &error)) {
        if (error) {
            LOGE("Failed to initialize GStreamer: %s", error->message);
            g_error_free(error);
        } else {
            LOGE("Failed to initialize GStreamer: Unknown error");
        }
        return JNI_FALSE;
    }
    
    LOGI("GStreamer initialized, registering plugins...");
    
    // Registrar plugins manualmente usando gst_plugin_register_static
    LOGI("Registering plugin: coreelements");
    gboolean result1 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "coreelements", "GStreamer core elements",
        gst_plugin_coreelements_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin coreelements registered: %s", result1 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: videoconvertscale");
    gboolean result2 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "videoconvertscale", "Video conversion and scaling",
        gst_plugin_videoconvertscale_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin videoconvertscale registered: %s", result2 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: x264");
    gboolean result3 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "x264", "x264 encoder",
        gst_plugin_x264_register,
        "1.28.2", "GPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin x264 registered: %s", result3 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: videoparsersbad");
    gboolean result4 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "videoparsersbad", "Video parsers",
        gst_plugin_videoparsersbad_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin videoparsersbad registered: %s", result4 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: rtp");
    gboolean result5 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "rtp", "RTP payloaders and depayloaders",
        gst_plugin_rtp_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin rtp registered: %s", result5 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: udp");
    gboolean result6 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "udp", "UDP sink and source",
        gst_plugin_udp_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin udp registered: %s", result6 ? "SUCCESS" : "FAILED");
    
    // PLUGIN ANDROIDMEDIA COMENTADO: Requiere libgstgl-1.0.a y libgstphotography-1.0.a
    // LOGI("Registering plugin: androidmedia");
    // gboolean result7 = gst_plugin_register_static(
    //     GST_VERSION_MAJOR, GST_VERSION_MINOR,
    //     "androidmedia", "Android media codecs",
    //     gst_plugin_androidmedia_register,
    //     "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    // );
    // LOGI("Plugin androidmedia registered: %s", result7 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: app");
    gboolean result7 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "app", "Application source and sink",
        gst_plugin_app_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin app registered: %s", result7 ? "SUCCESS" : "FAILED");
    
    LOGI("Registering plugin: videotestsrc");
    gboolean result8 = gst_plugin_register_static(
        GST_VERSION_MAJOR, GST_VERSION_MINOR,
        "videotestsrc", "Video test source",
        gst_plugin_videotestsrc_register,
        "1.28.2", "LGPL", "GStreamer", "GStreamer", "http://gstreamer.net/"
    );
    LOGI("Plugin videotestsrc registered: %s", result8 ? "SUCCESS" : "FAILED");
    
    gstreamer_initialized = TRUE;
    LOGI("GStreamer initialized successfully with plugins");
    
    // Imprimir versión de GStreamer
    guint major, minor, micro, nano;
    gst_version(&major, &minor, &micro, &nano);
    LOGI("GStreamer version: %u.%u.%u.%u", major, minor, micro, nano);
    
    // Listar plugins disponibles para debugging
    GstRegistry *registry = gst_registry_get();
    GList *plugins = gst_registry_get_plugin_list(registry);
    LOGI("Available plugins: %d", g_list_length(plugins));
    
    GList *l;
    int count = 0;
    for (l = plugins; l != NULL; l = l->next) {
        GstPlugin *plugin = GST_PLUGIN(l->data);
        LOGI("Plugin %d: %s", count++, gst_plugin_get_name(plugin));
        if (count >= 10) break; // Solo mostrar los primeros 10
    }
    
    gst_plugin_list_free(plugins);
    
    // Listar elementos disponibles
    GList *features = gst_registry_get_feature_list(registry, GST_TYPE_ELEMENT_FACTORY);
    LOGI("Available elements: %d", g_list_length(features));
    
    count = 0;
    for (l = features; l != NULL; l = l->next) {
        GstElementFactory *factory = GST_ELEMENT_FACTORY(l->data);
        LOGI("Element %d: %s", count++, gst_element_factory_get_longname(factory));
        if (count >= 10) break; // Solo mostrar los primeros 10
    }
    
    gst_plugin_feature_list_free(features);
    
    return JNI_TRUE;
}

// Callback para mensajes del bus
static gboolean bus_call(GstBus *bus, GstMessage *msg, gpointer data) {
    GStreamerPipeline *pipeline = (GStreamerPipeline *)data;
    
    switch (GST_MESSAGE_TYPE(msg)) {
        case GST_MESSAGE_EOS:
            LOGI("End of stream");
            gst_element_set_state(pipeline->pipeline, GST_STATE_NULL);
            break;
            
        case GST_MESSAGE_ERROR: {
            GError *err = NULL;
            gchar *debug = NULL;
            gst_message_parse_error(msg, &err, &debug);
            LOGE("Error: %s", err->message);
            g_error_free(err);
            g_free(debug);
            gst_element_set_state(pipeline->pipeline, GST_STATE_NULL);
            break;
        }
        
        case GST_MESSAGE_WARNING: {
            GError *err = NULL;
            gchar *debug = NULL;
            gst_message_parse_warning(msg, &err, &debug);
            LOGI("Warning: %s", err->message);
            g_error_free(err);
            g_free(debug);
            break;
        }
        
        case GST_MESSAGE_STATE_CHANGED: {
            GstState old_state, new_state, pending_state;
            gst_message_parse_state_changed(msg, &old_state, &new_state, &pending_state);
            LOGI("State changed: %s -> %s",
                 gst_element_state_get_name(old_state),
                 gst_element_state_get_name(new_state));
            break;
        }
        
        default:
            break;
    }
    
    return TRUE;
}

// Crear pipeline
jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeCreatePipeline(
    JNIEnv *env, jobject obj, jstring pipeline_str, jobject surface_view) {
    
    // Verificar que GStreamer esté inicializado
    if (!gstreamer_initialized) {
        LOGE("GStreamer not initialized! Call nativeInit() first");
        return 0;
    }
    
    const char *pipeline_cstr = env->GetStringUTFChars(pipeline_str, NULL);
    LOGI("Creating pipeline: %s", pipeline_cstr);
    
    GStreamerPipeline *pipeline = new GStreamerPipeline();
    memset(pipeline, 0, sizeof(GStreamerPipeline));
    
    GError *error = NULL;
    
    // Crear pipeline desde descripción de texto
    pipeline->pipeline = gst_parse_launch(pipeline_cstr, &error);
    
    if (error != NULL) {
        LOGE("Error creating pipeline: %s", error->message);
        g_error_free(error);
        env->ReleaseStringUTFChars(pipeline_str, pipeline_cstr);
        delete pipeline;
        return 0;
    }
    
    if (pipeline->pipeline == NULL) {
        LOGE("Pipeline is NULL after creation");
        env->ReleaseStringUTFChars(pipeline_str, pipeline_cstr);
        delete pipeline;
        return 0;
    }
    
    // Obtener elementos del pipeline
    pipeline->appsrc = gst_bin_get_by_name(GST_BIN(pipeline->pipeline), "videosrc");
    if (pipeline->appsrc == NULL) {
        LOGE("Failed to get appsrc element 'videosrc' from pipeline");
        gst_object_unref(pipeline->pipeline);
        env->ReleaseStringUTFChars(pipeline_str, pipeline_cstr);
        delete pipeline;
        return 0;
    }
    
    // Configurar appsrc
    g_object_set(G_OBJECT(pipeline->appsrc),
                 "stream-type", GST_APP_STREAM_TYPE_STREAM,
                 "format", GST_FORMAT_TIME,
                 "is-live", TRUE,
                 "do-timestamp", TRUE,
                 NULL);
    
    // Configurar caps para H.264
    GstCaps *caps = gst_caps_new_simple("video/x-h264",
                                        "stream-format", G_TYPE_STRING, "byte-stream",
                                        "alignment", G_TYPE_STRING, "au",
                                        NULL);
    g_object_set(G_OBJECT(pipeline->appsrc), "caps", caps, NULL);
    gst_caps_unref(caps);
    
    LOGI("appsrc configured successfully");
    
    pipeline->camerabin = NULL;  // No usado con appsrc
    pipeline->videoscale = NULL;
    pipeline->x264enc = NULL;
    pipeline->h264parse = NULL;
    pipeline->rtph264pay = NULL;
    pipeline->udpsink = NULL;
    
    // Inicializar otros campos
    pipeline->is_running = FALSE;
    pipeline->dropped_frames = 0;
    pipeline->latency = 0;
    
    // Configurar bus para mensajes
    pipeline->bus = gst_element_get_bus(pipeline->pipeline);
    if (pipeline->bus != NULL) {
        pipeline->bus_watch_id = gst_bus_add_watch(pipeline->bus, bus_call, pipeline);
        gst_object_unref(pipeline->bus);
    } else {
        LOGE("Failed to get bus from pipeline");
    }
    
    // Configurar SurfaceView para preview (solo si se proporciona)
    // NOTA: Para pipelines sin preview (como videotestsrc ! fakesink), 
    // surface_view será NULL y esto se omite correctamente
    if (surface_view != NULL) {
        // TODO: Implementar preview cuando sea necesario
        // Por ahora, los pipelines de prueba no requieren preview
        LOGI("SurfaceView provided but preview not implemented yet");
    }
    
    LOGI("Pipeline created successfully");
    env->ReleaseStringUTFChars(pipeline_str, pipeline_cstr);
    
    return (jlong)pipeline;
}

// Iniciar pipeline
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeStartPipeline(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("Invalid pipeline handle");
        return;
    }
    
    GstStateChangeReturn ret = gst_element_set_state(pipeline->pipeline, GST_STATE_PLAYING);
    
    if (ret == GST_STATE_CHANGE_FAILURE) {
        LOGE("Failed to set pipeline to PLAYING state");
        return;
    }
    
    pipeline->is_running = TRUE;
    LOGI("Pipeline started");
}

// Detener pipeline
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeStopPipeline(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("Invalid pipeline handle");
        return;
    }
    
    gst_element_set_state(pipeline->pipeline, GST_STATE_NULL);
    pipeline->is_running = FALSE;
    LOGI("Pipeline stopped");
}

// Destruir pipeline
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeDestroyPipeline(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("Invalid pipeline handle");
        return;
    }
    
    // Detener pipeline
    gst_element_set_state(pipeline->pipeline, GST_STATE_NULL);
    
    // Remover bus watch
    if (pipeline->bus_watch_id > 0) {
        g_source_remove(pipeline->bus_watch_id);
    }
    
    // Liberar referencias
    if (pipeline->camerabin) gst_object_unref(pipeline->camerabin);
    if (pipeline->videoscale) gst_object_unref(pipeline->videoscale);
    if (pipeline->x264enc) gst_object_unref(pipeline->x264enc);
    if (pipeline->h264parse) gst_object_unref(pipeline->h264parse);
    if (pipeline->rtph264pay) gst_object_unref(pipeline->rtph264pay);
    if (pipeline->udpsink) gst_object_unref(pipeline->udpsink);
    if (pipeline->pipeline) gst_object_unref(pipeline->pipeline);
    
    delete pipeline;
    LOGI("Pipeline destroyed");
}

// Pausar pipeline
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativePausePipeline(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("Invalid pipeline handle");
        return;
    }
    
    gst_element_set_state(pipeline->pipeline, GST_STATE_PAUSED);
    LOGI("Pipeline paused");
}

// Reanudar pipeline
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeResumePipeline(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("Invalid pipeline handle");
        return;
    }
    
    gst_element_set_state(pipeline->pipeline, GST_STATE_PLAYING);
    LOGI("Pipeline resumed");
}

// Obtener FPS
jdouble Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetFps(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL || !pipeline->is_running) {
        return 0.0;
    }
    
    // Obtener estadísticas del elemento x264enc
    if (pipeline->x264enc) {
        GstStructure *stats = NULL;
        g_object_get(G_OBJECT(pipeline->x264enc), "stats", &stats, NULL);
        
        if (stats) {
            gint fps = 0;
            gst_structure_get_int(stats, "fps", &fps);
            gst_structure_free(stats);
            return (jdouble)fps;
        }
    }
    
    return 0.0;
}

// Obtener bitrate
jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetBitrate(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL || !pipeline->is_running) {
        return 0;
    }
    
    // Obtener bitrate del elemento x264enc
    if (pipeline->x264enc) {
        guint bitrate = 0;
        g_object_get(G_OBJECT(pipeline->x264enc), "bitrate", &bitrate, NULL);
        return (jlong)bitrate * 1000; // Convertir a bps
    }
    
    return 0;
}

// Obtener frames descartados
jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetDroppedFrames(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL || !pipeline->is_running) {
        return 0;
    }
    
    // Obtener estadísticas del pipeline
    GstQuery *query = gst_query_new_latency();
    
    if (gst_element_query(pipeline->pipeline, query)) {
        GstClockTime min_latency = 0, max_latency = 0;
        gboolean live = FALSE;
        gst_query_parse_latency(query, &live, &min_latency, &max_latency);
        pipeline->latency = (gint64)(min_latency / GST_MSECOND);
    }
    
    gst_query_unref(query);
    
    return pipeline->dropped_frames;
}

// Obtener latencia
jlong Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativeGetLatency(
    JNIEnv *env, jobject obj, jlong handle) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL || !pipeline->is_running) {
        return 0;
    }
    
    GstQuery *query = gst_query_new_latency();
    
    if (gst_element_query(pipeline->pipeline, query)) {
        GstClockTime min_latency, max_latency;
        gboolean live;
        gst_query_parse_latency(query, &live, &min_latency, &max_latency);
        pipeline->latency = min_latency / GST_MSECOND;
    }
    
    gst_query_unref(query);
    
    return pipeline->latency;
}

// Enviar datos H.264 al appsrc
void Java_com_mentor_mobile_gstreamer_GStreamerPipeline_nativePushH264Data(
    JNIEnv *env, jobject obj, jlong handle, jbyteArray data, jint size, jlong timestamp) {
    
    GStreamerPipeline *pipeline = (GStreamerPipeline *)handle;
    
    if (pipeline == NULL) {
        LOGE("nativePushH264Data: pipeline is NULL");
        return;
    }
    
    if (!pipeline->is_running) {
        LOGE("nativePushH264Data: pipeline is not running");
        return;
    }
    
    if (pipeline->appsrc == NULL) {
        LOGE("nativePushH264Data: appsrc is NULL - pipeline may not be created correctly");
        return;
    }
    
    if (size <= 0 || data == NULL) {
        LOGE("nativePushH264Data: Invalid data (size=%d, data=%p)", size, data);
        return;
    }
    
    // Log menos frecuente para evitar spam
    static int frame_count = 0;
    frame_count++;
    if (frame_count % 30 == 0) { // Log cada 30 frames (1 segundo a 30fps)
        LOGI("nativePushH264Data: Frame %d, size=%d bytes, timestamp=%lld", frame_count, size, timestamp);
    }
    
    // Obtener datos del array de Java
    jbyte *dataPtr = env->GetByteArrayElements(data, NULL);
    if (dataPtr == NULL) {
        LOGE("nativePushH264Data: Failed to get byte array elements");
        return;
    }
    
    // Crear buffer de GStreamer
    GstBuffer *buffer = gst_buffer_new_allocate(NULL, size, NULL);
    if (buffer == NULL) {
        LOGE("nativePushH264Data: Failed to allocate GStreamer buffer");
        env->ReleaseByteArrayElements(data, dataPtr, JNI_ABORT);
        return;
    }
    
    GstMapInfo map;
    if (!gst_buffer_map(buffer, &map, GST_MAP_WRITE)) {
        LOGE("nativePushH264Data: Failed to map buffer");
        gst_buffer_unref(buffer);
        env->ReleaseByteArrayElements(data, dataPtr, JNI_ABORT);
        return;
    }
    
    memcpy(map.data, dataPtr, size);
    gst_buffer_unmap(buffer, &map);
    
    // Establecer timestamp
    GST_BUFFER_PTS(buffer) = timestamp * GST_USECOND;
    GST_BUFFER_DTS(buffer) = timestamp * GST_USECOND;
    
    // Enviar buffer al appsrc
    GstFlowReturn ret = gst_app_src_push_buffer(GST_APP_SRC(pipeline->appsrc), buffer);
    
    if (ret != GST_FLOW_OK) {
        LOGE("Error pushing buffer to appsrc: %d (%s)", ret, 
             ret == GST_FLOW_FLUSHING ? "FLUSHING" :
             ret == GST_FLOW_EOS ? "EOS" :
             ret == GST_FLOW_NOT_LINKED ? "NOT_LINKED" :
             ret == GST_FLOW_ERROR ? "ERROR" : "UNKNOWN");
    } else if (frame_count % 30 == 0) {
        LOGI("Buffer pushed successfully to appsrc");
    }
    
    // Liberar array de Java
    env->ReleaseByteArrayElements(data, dataPtr, JNI_ABORT);
}
