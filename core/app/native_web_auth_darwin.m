//go:build darwin && cgo

#import <AuthenticationServices/AuthenticationServices.h>
#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <stdlib.h>
#import <string.h>

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wunguarded-availability-new"

extern void aivoASWebAuthComplete(long long sessionID, char *callbackURL, char *errorMessage);

@interface AivoWebAuthPresentationContextProvider : NSObject <ASWebAuthenticationPresentationContextProviding>
@end

@implementation AivoWebAuthPresentationContextProvider

- (ASPresentationAnchor)presentationAnchorForWebAuthenticationSession:(ASWebAuthenticationSession *)session {
    NSWindow *window = NSApplication.sharedApplication.keyWindow;
    if (window == nil) {
        window = NSApplication.sharedApplication.mainWindow;
    }
    if (window == nil) {
        window = NSApplication.sharedApplication.windows.firstObject;
    }
    return window;
}

@end

static NSMutableDictionary<NSNumber *, ASWebAuthenticationSession *> *AivoWebAuthSessions;
static AivoWebAuthPresentationContextProvider *AivoWebAuthContextProvider;

static char *AivoCopyCString(NSString *value) {
    if (value == nil) {
        return NULL;
    }
    return strdup(value.UTF8String);
}

int AivoStartASWebAuthenticationSession(const char *authURL, const char *callbackScheme, long long sessionID) {
    if (@available(macOS 10.15, *)) {
        NSString *authURLString = [NSString stringWithUTF8String:authURL];
        NSURL *url = [NSURL URLWithString:authURLString];
        if (url == nil) {
            aivoASWebAuthComplete(sessionID, NULL, AivoCopyCString(@"Invalid authentication URL."));
            return 0;
        }

        NSString *scheme = nil;
        if (callbackScheme != NULL && strlen(callbackScheme) > 0) {
            scheme = [NSString stringWithUTF8String:callbackScheme];
        }

        dispatch_async(dispatch_get_main_queue(), ^{
            if (AivoWebAuthSessions == nil) {
                AivoWebAuthSessions = [NSMutableDictionary dictionary];
            }
            if (AivoWebAuthContextProvider == nil) {
                AivoWebAuthContextProvider = [AivoWebAuthPresentationContextProvider new];
            }

            NSNumber *key = @(sessionID);
            ASWebAuthenticationSession *session = [[ASWebAuthenticationSession alloc]
                initWithURL:url
                callbackURLScheme:scheme
                completionHandler:^(NSURL *callbackURL, NSError *error) {
                    char *callbackCString = AivoCopyCString(callbackURL.absoluteString);
                    char *errorCString = AivoCopyCString(error.localizedDescription);
                    [AivoWebAuthSessions removeObjectForKey:key];
                    aivoASWebAuthComplete(sessionID, callbackCString, errorCString);
                }];
            session.presentationContextProvider = AivoWebAuthContextProvider;
            session.prefersEphemeralWebBrowserSession = NO;
            AivoWebAuthSessions[key] = session;

            if (![session start]) {
                [AivoWebAuthSessions removeObjectForKey:key];
                aivoASWebAuthComplete(sessionID, NULL, AivoCopyCString(@"Unable to start macOS authentication session."));
            }
        });
        return 1;
    }
    aivoASWebAuthComplete(sessionID, NULL, AivoCopyCString(@"ASWebAuthenticationSession requires macOS 10.15 or later."));
    return 0;
}

void AivoCancelASWebAuthenticationSession(long long sessionID) {
    if (@available(macOS 10.15, *)) {
        dispatch_async(dispatch_get_main_queue(), ^{
            NSNumber *key = @(sessionID);
            ASWebAuthenticationSession *session = AivoWebAuthSessions[key];
            if (session != nil) {
                [session cancel];
                [AivoWebAuthSessions removeObjectForKey:key];
            }
        });
    }
}

#pragma clang diagnostic pop
