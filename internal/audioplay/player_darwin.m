//go:build darwin && cgo
#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>
void *elabAudioPlay(const void *bytes,int length) {
 @autoreleasepool {
  NSData *data=[NSData dataWithBytes:bytes length:(NSUInteger)length];
  AVAudioPlayer *player=[[AVAudioPlayer alloc] initWithData:data error:nil];
  if(!player) return NULL;
  player.volume=1.0;
  if(![player prepareToPlay] || ![player play]) { [player release];return NULL; }
  return player;
 }
}
void elabAudioStop(void *handle) {
 @autoreleasepool { AVAudioPlayer *player=(AVAudioPlayer *)handle;[player stop];[player release]; }
}
